package populate

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yamls"

	corev1 "k8s.io/api/core/v1"

	v1 "github.com/jenkins-x-plugins/jx-secret/pkg/apis/external/v1"
	"github.com/jenkins-x-plugins/jx-secret/pkg/apis/mapping/v1alpha1"
	"github.com/jenkins-x-plugins/jx-secret/pkg/cmd/vault/wait"
	"github.com/jenkins-x-plugins/jx-secret/pkg/extsecrets"
	"github.com/jenkins-x-plugins/jx-secret/pkg/extsecrets/editor"
	"github.com/jenkins-x-plugins/jx-secret/pkg/extsecrets/secretfacade"
	"github.com/jenkins-x-plugins/jx-secret/pkg/rootcmd"
	"github.com/jenkins-x-plugins/jx-secret/pkg/schemas/generators"
	"github.com/jenkins-x-plugins/jx-secret/pkg/vaults"
	"github.com/jenkins-x-plugins/jx-secret/pkg/vaults/vaultcli"
	"github.com/jenkins-x-plugins/secretfacade/pkg/secretstore"
	jxcore "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-kube-client/v3/pkg/kubeclient"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	k8swait "k8s.io/apimachinery/pkg/util/wait"
)

var (
	cmdLong = templates.LongDesc(`
		Populates any missing secret values which can be automatically generated or that have default values"
`)

	cmdExample = templates.Examples(`
		%s populate
	`)

	DefaultBackoff = k8swait.Backoff{
		Steps:    5,
		Duration: 2 * time.Second,
		Factor:   2.0,
		Jitter:   0.1,
	}
)

// Options the options for the command
type Options struct {
	secretfacade.Options
	WaitDuration        time.Duration
	HelmSecretFolder    string
	Backoff             *k8swait.Backoff
	Results             []*secretfacade.SecretPair
	CommandRunner       cmdrunner.CommandRunner
	QuietCommandRunner  cmdrunner.CommandRunner
	NoWait              bool
	DisableLoadResults  bool
	Generators          map[string]generators.Generator
	HelmSecretValues    map[string]map[string]string
	Requirements        *jxcore.RequirementsConfig
	BootSecretNamespace string
	DisableSecretFolder bool
}

// NewCmdPopulate creates a command object for the command
func NewCmdPopulate() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "populate",
		Short:   "Populates any missing secret values which can be automatically generated, generated using a template or that have default values",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName),
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.Run()
		},
	}
	cmd.Flags().StringVarP(&o.Namespace, "ns", "n", "", "the namespace to filter the ExternalSecret resources")
	cmd.Flags().StringVarP(&o.BootSecretNamespace, "boot-secret-namespace", "", "", "the namespace to that contains the boot secret used to populate git secrets from")
	cmd.Flags().StringVarP(&o.HelmSecretFolder, "helm-secrets-dir", "", "", "the directory where the helm secrets live with a folder per namespace and a file with a '.yaml' extension for each secret name. Defaults to $JX_HELM_SECRET_FOLDER")
	cmd.Flags().StringVarP(&o.Dir, "dir", "d", ".", "the directory to look for the .jx/secret/mapping/secret-mappings.yaml file")
	cmd.Flags().BoolVarP(&o.NoWait, "no-wait", "", false, "disables waiting for the secret store (e.g. vault) to be available")
	cmd.Flags().DurationVarP(&o.WaitDuration, "wait", "w", 2*time.Hour, "the maximum time period to wait for the vault pod to be ready if using the vault backendType")
	cmd.Flags().StringVarP(&o.SecretNamespace, "secret-namespace", "", vaults.DefaultVaultNamespace, "the namespace in which secret infrastructure resides such as Hashicorp Vault")

	o.Options.AddFlags(cmd)
	return cmd, o
}

func (o *Options) Validate() error {
	err := o.Options.Validate()
	if err != nil {
		return errors.Wrap(err, "error validating options")
	}

	if o.HelmSecretFolder == "" {
		o.HelmSecretFolder = extsecrets.DefaultHelmSecretFolder()
	}
	log.Logger().Debugf("loading default Secret data from helm secret folder: %s", o.HelmSecretFolder)

	if o.Backoff == nil {
		o.Backoff = &DefaultBackoff
	}
	return nil
}

// Run implements the command
func (o *Options) Run() error {
	err := o.Validate()
	if err != nil {
		return errors.Wrap(err, "error validating options")
	}

	if !o.DisableLoadResults {
		// get a list of external secrets which do not have corresponding k8s secret data populated
		results, err := o.VerifyAndFilter()
		if err != nil {
			return errors.Wrap(err, "failed to verify secrets")
		}
		o.Results = results
	}

	results := o.Results
	if len(results) == 0 {
		log.Logger().Infof("the %d ExternalSecrets are %s", len(o.ExternalSecrets), termcolor.ColorInfo("populated"))
		return nil
	}
	o.loadGenerators()

	waited := map[string]bool{}

	err = o.PopulateLoop(results, waited)
	if err != nil {
		return errors.Wrapf(err, "failed to populate secrets")
	}

	// lets run the loop again for any template / generators which need mandatory secrets as inputs
	results, err = o.VerifyAndFilter()
	if err != nil {
		return errors.Wrap(err, "failed to verify secrets on second pass")
	}
	o.Results = results
	if len(results) == 0 {
		log.Logger().Infof("the %d ExternalSecrets on second pass are %s", len(o.ExternalSecrets), termcolor.ColorInfo("populated"))
		return nil
	}
	err = o.PopulateLoop(results, waited)
	if err != nil {
		return errors.Wrapf(err, "failed to populate secrets on second pass")
	}
	return nil
}

// PopulateLoop populates any external secret stores
func (o *Options) PopulateLoop(results []*secretfacade.SecretPair, waited map[string]bool) error {
	for _, r := range results {
		name := r.ExternalSecret.Name
		backendType := r.ExternalSecret.Spec.BackendType

		localReplica := false
		if backendType == "local" {
			ann := r.ExternalSecret.Annotations
			if ann != nil {
				// ignore local replicas
				if ann[extsecrets.ReplicaAnnotation] == "true" {
					continue
				}
				if ann[extsecrets.ReplicateToAnnotation] != "" {
					localReplica = true
				}
			}
		}

		// lets wait until the backend is available
		if !waited[backendType] {
			err := o.waitForBackend(backendType)
			if err != nil {
				return errors.Wrapf(err, "failed to wait for backend type %s", backendType)
			}
			waited[backendType] = true
		}

		secretManager, err := o.getSecretManager(backendType)

		if err != nil {
			return errors.Wrapf(err, "failed to create a secret manager for ExternalSecret %s", name)
		}

		data := r.ExternalSecret.Spec.Data
		m := map[string]*editor.KeyProperties{}
		newValueMap := map[string]bool{}
		for i := range data {
			d := &data[i]
			key := GetSecretKey(v1alpha1.BackendType(backendType), r.ExternalSecret.Name, d.Key)
			property := d.Property
			entryName := d.Name
			keyProperties := m[key]
			if keyProperties == nil {
				keyProperties = &editor.KeyProperties{
					Key: key,
				}
				if r.ExternalSecret.Spec.BackendType == string(v1alpha1.BackendTypeGSM) {
					if r.ExternalSecret.Spec.ProjectID != "" {
						keyProperties.GCPProject = r.ExternalSecret.Spec.ProjectID
					} else {
						log.Logger().Warnf("no GCP project ID found for external secret %s, defaulting to current project", r.ExternalSecret.Name)
					}
				}

				m[key] = keyProperties
			}

			currentValue := ""
			if r.Secret != nil && r.Secret.Data != nil {
				currentValue = string(r.Secret.Data[d.Name])
			}
			var value string
			value, err = o.generateSecretValue(r, name, d.Name, currentValue)
			if err != nil {
				return errors.Wrapf(err, "failed to ask user secret value property %s for key %s on ExternalSecret %s", property, key, name)
			}

			// lets always update values for local replicas so that replication triggers to other namespaces
			if value != "" && (value != currentValue || localReplica) {
				newValueMap[key] = true
			}
			if value == "" {
				value = currentValue
			}

			// lets always modify all entries if there is a new value
			// as back ends like vault can't handle only writing 1 value
			keyProperties.Properties = append(keyProperties.Properties, editor.PropertyValue{
				Property: property,
				Value:    value,
				Name:     entryName,
			})
		}
		for key, keyProperties := range m {
			if newValueMap[key] && len(keyProperties.Properties) > 0 {
				annotations := r.ExternalSecret.Spec.Template.Metadata.Annotations

				// handle replicate to annotation for local secrets so that we also copy the secret to other namespaces
				replicateTo := ""
				if r.ExternalSecret.Annotations != nil {
					replicateTo = r.ExternalSecret.Annotations[extsecrets.ReplicateToAnnotation]
				}
				if replicateTo != "" {
					annotations[extsecrets.ReplicateToAnnotation] = replicateTo
				}

				labels := r.ExternalSecret.Spec.Template.Metadata.Labels
				secretType := corev1.SecretType(r.ExternalSecret.Spec.Template.Type)
				sv := CreateSecretValue(v1alpha1.BackendType(r.ExternalSecret.Spec.BackendType), keyProperties.Properties, annotations, labels, secretType)
				err = secretManager.SetSecret(GetExternalSecretLocation(&r.ExternalSecret), GetSecretKey(v1alpha1.BackendType(r.ExternalSecret.Spec.BackendType), r.ExternalSecret.Name, key), &sv)
				if err != nil {
					return errors.Wrapf(err, "failed to save properties %s on ExternalSecret %s", keyProperties.String(), name)
				}
			}
		}
	}
	return nil
}

func GetSecretStore(backendType v1alpha1.BackendType) secretstore.SecretStoreType {
	switch backendType {
	case v1alpha1.BackendTypeLocal:
		return secretstore.SecretStoreTypeKubernetes
	default:
		return secretstore.SecretStoreType(backendType)
	}
}

func GetSecretKey(backendType v1alpha1.BackendType, externalSecretName string, keyName string) string {
	if backendType == v1alpha1.BackendTypeLocal {
		return externalSecretName
	}
	return keyName
}

func CreateSecretValue(backendType v1alpha1.BackendType, values []editor.PropertyValue, annotations map[string]string, labels map[string]string, secretType corev1.SecretType) secretstore.SecretValue {
	formatValues := func(values []editor.PropertyValue) map[string]string {
		properties := map[string]string{}
		for _, p := range values {
			propertyName := p.Property
			if propertyName == "" {
				propertyName = p.Name
			}
			properties[propertyName] = p.Value
		}
		return properties
	}

	switch backendType {
	case v1alpha1.BackendTypeVault:
		return secretstore.SecretValue{PropertyValues: formatValues(values)}
	case v1alpha1.BackendTypeLocal:
		sv := secretstore.SecretValue{PropertyValues: formatValues(values)}
		sv.Labels = labels
		sv.Annotations = annotations
		sv.SecretType = secretType
		return sv
	case v1alpha1.BackendTypeAWSSecretsManager:
		return secretstore.SecretValue{PropertyValues: formatValues(values)}
	default:
		if len(values) == 1 && values[0].Property == "" {
			return secretstore.SecretValue{Value: values[0].Value}
		}
		return secretstore.SecretValue{PropertyValues: formatValues(values)}
	}
}

func GetExternalSecretLocation(extsec *v1.ExternalSecret) string {
	switch v1alpha1.BackendType(extsec.Spec.BackendType) {
	case v1alpha1.BackendTypeGSM:
		return extsec.Spec.ProjectID
	case v1alpha1.BackendTypeAzure:
		return extsec.Spec.KeyVaultName
	case v1alpha1.BackendTypeVault:
		return os.Getenv("VAULT_ADDR")
	case v1alpha1.BackendTypeAWSSecretsManager:
		return extsec.Spec.Region
	case v1alpha1.BackendTypeLocal:
		return extsec.Namespace
	}
	return ""
}

func (o *Options) generateSecretValue(s *secretfacade.SecretPair, secretName, property, currentValue string) (string, error) {
	object, err := s.SchemaObject()
	if err != nil {
		return "", errors.Wrapf(err, "failed to find object schema for object %s property %s", secretName, property)
	}
	if object == nil {
		return o.helmSecretValue(s, property)
	}
	propertySchema := object.FindProperty(property)
	if propertySchema == nil {
		return o.helmSecretValue(s, property)
	}

	templateText := propertySchema.Template
	if templateText != "" {
		// don't regenerate if configured to only do so if non blank
		if propertySchema.OnlyTemplateIfBlank && currentValue != "" {
			return "", nil
		}
		return o.EvaluateTemplate(s.ExternalSecret.Namespace, secretName, property, templateText, propertySchema.Retry)
	}

	// for now don't regenerate if we have a current value
	// longer term we could maybe use metadata to decide how frequently to run generators or regenerate if the value is too old etc
	if currentValue != "" {
		return "", nil
	}

	generatorName := propertySchema.Generator
	if generatorName == "" {
		if propertySchema.DefaultValue != "" {
			return propertySchema.DefaultValue, nil
		}

		// lets try fetch the default value from the generated helm secrets
		return o.helmSecretValue(s, property)
	}

	generator := o.Generators[generatorName]
	if generator == nil {
		return "", errors.Errorf("could not find generator %s for property %s in object %s", generatorName, property, secretName)
	}

	args := &generators.Arguments{
		Object:   object,
		Property: propertySchema,
	}
	value, err := generator(args)
	if err != nil {
		return value, errors.Wrapf(err, "failed to invoke generator %s for property %s in object %s", generatorName, property, secretName)
	}
	return value, nil
}

func (o *Options) waitForBackend(backendType string) error {
	if backendType != "vault" {
		return nil
	}
	if o.NoWait {
		log.Logger().Infof("disabling waiting for vault pod to be ready")
		return nil
	}

	_, wo := wait.NewCmdWait()
	wo.WaitDuration = o.WaitDuration
	wo.KubeClient = o.KubeClient
	wo.Namespace = o.SecretNamespace

	err := wo.Run()
	if err != nil {
		return errors.Wrapf(err, "failed to wait for vault backend")
	}
	return nil
}

func (o *Options) loadGenerators() {
	if o.Generators == nil {
		o.Generators = map[string]generators.Generator{}
	}
	o.Generators["hmac"] = generators.Hmac
	o.Generators["password"] = generators.Password
	ns := o.BootSecretNamespace
	if ns == "" {
		var err error
		ns, err = kubeclient.CurrentNamespace()
		if err != nil {
			log.Logger().Warnf("failed to get current namespace, defaulting to jx: %s", err.Error())
		}
		if ns == "" {
			ns = "jx"
		}
	}
	o.Generators["gitOperator.username"] = generators.SecretEntry(o.KubeClient, ns, "jx-boot", "username")
	o.Generators["gitOperator.password"] = generators.SecretEntry(o.KubeClient, ns, "jx-boot", "password")
}

func (o *Options) getSecretManager(backendType string) (secretstore.Interface, error) {
	store := GetSecretStore(v1alpha1.BackendType(backendType))

	isExternalVault := os.Getenv("EXTERNAL_VAULT")
	if isExternalVault == "true" {
		log.Logger().Infof("connecting to external vault")
	}
	if store == secretstore.SecretStoreTypeVault && isExternalVault != "true" {
		envMap, err := vaultcli.CreateVaultEnv(o.KubeClient)
		if err != nil {
			return nil, errors.Wrapf(err, "error creating vault env vars")
		}
		for k, v := range envMap {
			err := os.Setenv(k, v)
			if err != nil {
				return nil, errors.Wrapf(err, "failed setting env var %s for vault auth", k)
			}
		}
	}
	secretManager, err := o.SecretStoreManagerFactory.NewSecretManager(store)
	if err != nil {
		return nil, errors.Wrapf(err, "error creating secret manager")
	}
	return secretManager, nil
}

func (o *Options) helmSecretValue(s *secretfacade.SecretPair, entryName string) (string, error) {
	ns := s.Namespace()
	name := s.Name()
	key := scm.Join(ns, name)
	if o.HelmSecretValues == nil {
		o.HelmSecretValues = map[string]map[string]string{}
	}
	values := o.HelmSecretValues[key]
	if values == nil && !o.DisableSecretFolder {
		path := filepath.Join(o.HelmSecretFolder, ns, name+".yaml")

		exists, err := files.FileExists(path)
		if err != nil {
			return "", errors.Wrapf(err, "failed to check for file %s", path)
		}

		// lets populate an empty map so we don't try load this file again for another entry...
		o.HelmSecretValues[key] = map[string]string{}

		if !exists {
			log.Logger().Warnf("no helm secrets file %s exists so cannot default the external secret store data", path)
			return "", nil
		}

		secret := &corev1.Secret{}
		err = yamls.LoadFile(path, secret)
		if err != nil {
			return "", errors.Wrapf(err, "failed to parse Secret file %s", path)
		}

		values = map[string]string{}
		for k, v := range secret.Data {
			values[k] = string(v)
		}
		for k, v := range secret.StringData {
			values[k] = v
		}
		o.HelmSecretValues[key] = values
	}
	return values[entryName], nil
}

func KubectlExecRunner(podName string, sidecar string, runner cmdrunner.CommandRunner) func(c *cmdrunner.Command) (string, error) {
	return func(c *cmdrunner.Command) (string, error) {
		kc := *c
		kc.Name = "kubectl"
		kc.Args = append([]string{"exec", podName, "-t", "-c", sidecar, "--", c.Name}, c.Args...)
		return runner(&kc)
	}
}
