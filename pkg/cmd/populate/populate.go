package populate

import (
	"fmt"
	"os"
	"time"

	jxcore "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-kube-client/v3/pkg/kubeclient"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/jenkins-x/jx-secret/pkg/apis/mapping/v1alpha1"
	"github.com/jenkins-x/jx-secret/pkg/cmd/vault/wait"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets/editor"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets/editor/factory"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets/secretfacade"
	"github.com/jenkins-x/jx-secret/pkg/rootcmd"
	"github.com/jenkins-x/jx-secret/pkg/schemas/generators"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	cmdLong = templates.LongDesc(`
		Populates any missing secret values which can be automatically generated or that have default values"
`)

	cmdExample = templates.Examples(`
		%s populate
	`)
)

// Options the options for the command
type Options struct {
	secretfacade.Options
	WaitDuration        time.Duration
	Results             []*secretfacade.SecretPair
	CommandRunner       cmdrunner.CommandRunner
	QuietCommandRunner  cmdrunner.CommandRunner
	NoWait              bool
	Generators          map[string]generators.Generator
	Requirements        *jxcore.RequirementsConfig
	BootSecretNamespace string
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
	cmd.Flags().StringVarP(&o.Dir, "dir", "d", ".", "the directory to look for the .jx/secret/mapping/secret-mappings.yaml file")
	cmd.Flags().BoolVarP(&o.NoWait, "no-wait", "", false, "disables waiting for the secret store (e.g. vault) to be available")
	cmd.Flags().DurationVarP(&o.WaitDuration, "wait", "w", 2*time.Hour, "the maximum time period to wait for the vault pod to be ready if using the vault backendType")
	return cmd, o
}

// Run implements the command
func (o *Options) Run() error {
	// get a list of external secrets which do not have corresponding k8s secret data populated
	results, err := o.VerifyAndFilter()
	if err != nil {
		return errors.Wrap(err, "failed to verify secrets")
	}
	o.Results = results

	if len(results) == 0 {
		log.Logger().Infof("the %d ExternalSecrets are %s", len(o.ExternalSecrets), termcolor.ColorInfo("populated"))
		return nil
	}
	o.loadGenerators()

	waited := map[string]bool{}

	err = o.populateLoop(results, waited)
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
	err = o.populateLoop(results, waited)
	if err != nil {
		return errors.Wrapf(err, "failed to populate secrets on second pass")
	}
	return nil
}

func (o *Options) populateLoop(results []*secretfacade.SecretPair, waited map[string]bool) error {
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

		runner, quietRunner, err := o.secretCommandRunner(backendType)
		if err != nil {
			return errors.Wrapf(err, "failed to get command runner")
		}

		secEditor, err := factory.NewEditor(o.EditorCache, &r.ExternalSecret, runner, quietRunner, o.KubeClient)
		if err != nil {
			return errors.Wrapf(err, "failed to create a secret editor for ExternalSecret %s", name)
		}

		data := r.ExternalSecret.Spec.Data
		m := map[string]*editor.KeyProperties{}
		newValueMap := map[string]bool{}
		for i := range data {
			d := &data[i]
			key := d.Key
			property := d.Property
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
			})
		}
		for key, keyProperties := range m {
			if newValueMap[key] && len(keyProperties.Properties) > 0 {
				err = secEditor.Write(keyProperties)
				if err != nil {
					return errors.Wrapf(err, "failed to save properties %s on ExternalSecret %s", keyProperties.String(), name)
				}
			}
		}
	}
	return nil
}

func (o *Options) generateSecretValue(s *secretfacade.SecretPair, secretName, property, currentValue string) (string, error) {
	object, err := s.SchemaObject()
	if err != nil {
		return "", errors.Wrapf(err, "failed to find object schema for object %s property %s", secretName, property)
	}
	if object == nil {
		return "", nil
	}
	propertySchema := object.FindProperty(property)
	if propertySchema == nil {
		return "", nil
	}

	templateText := propertySchema.Template
	if templateText != "" {
		return o.EvaluateTemplate(s.ExternalSecret.Namespace, secretName, property, templateText)
	}

	// for now don't regenerate if we have a current value
	// longer term we could maybe use metadata to decide how frequently to run generators or regenerate if the value is too old etc
	if currentValue != "" {
		return "", nil
	}

	generatorName := propertySchema.Generator
	if generatorName == "" {
		return propertySchema.DefaultValue, nil
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

// secretCommandRunner should we use `kubectl exec` into a side car to execute the commands?
// if we are in the boot job we should
func (o *Options) secretCommandRunner(_ string) (cmdrunner.CommandRunner, cmdrunner.CommandRunner, error) {
	podName := os.Getenv("POD_NAME")
	if podName == "" {
		podName = os.Getenv("HOSTNAME")
	}
	sidecar := os.Getenv("JX_SECRET_SIDECAR")
	if o.CommandRunner == nil {
		o.CommandRunner = cmdrunner.DefaultCommandRunner
	}
	if o.QuietCommandRunner == nil {
		o.QuietCommandRunner = cmdrunner.QuietCommandRunner
	}
	if sidecar == "" || podName == "" {
		return o.CommandRunner, o.QuietCommandRunner, nil
	}
	return KubectlExecRunner(podName, sidecar, o.CommandRunner), KubectlExecRunner(podName, sidecar, o.QuietCommandRunner), nil
}

func KubectlExecRunner(podName string, sidecar string, runner cmdrunner.CommandRunner) func(c *cmdrunner.Command) (string, error) {
	return func(c *cmdrunner.Command) (string, error) {
		kc := *c
		kc.Name = "kubectl"
		kc.Args = append([]string{"exec", podName, "-t", "-c", sidecar, "--", c.Name}, c.Args...)
		return runner(&kc)
	}
}
