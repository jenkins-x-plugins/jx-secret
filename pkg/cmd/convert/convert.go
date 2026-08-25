package convert

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jenkins-x/jx-helpers/v3/pkg/options"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/jenkins-x-plugins/jx-secret/pkg/cmd/convert/edit"
	"github.com/jenkins-x-plugins/jx-secret/pkg/extsecrets"
	"github.com/jenkins-x-plugins/jx-secret/pkg/schemas"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"

	"github.com/jenkins-x-plugins/jx-secret/pkg/apis/mapping/v1alpha1"
	schema "github.com/jenkins-x-plugins/jx-secret/pkg/apis/schema/v1alpha1"
	"github.com/jenkins-x-plugins/jx-secret/pkg/rootcmd"
	"github.com/jenkins-x-plugins/jx-secret/pkg/secretmapping"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kyamls"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var (
	info = termcolor.ColorInfo

	labelLong = templates.LongDesc(`
		Converts all Secret resources in the path to ExternalSecret resources so they can be checked into git
`)

	labelExample = templates.Examples(`
		# converts all the Secret resources into ExternalSecret resources so they can be checked into git
		%s convert --source-dir=config-root
	`)
)

// LabelOptions the options for the command
type Options struct {
	options.BaseOptions
	kyamls.Filter

	Dir              string `env:"JX_DIR"`
	DefaultNamespace string `env:"JX_DEFAULT_NAMESPACE"`
	SourceDir        string
	VersionStreamDir string
	HelmSecretFolder string
	SecretMapping    *v1alpha1.SecretMapping

	Prefix string

	warnedAlicloud bool
}

// NewCmdSecretConvert creates a command object for the command
func NewCmdSecretConvert() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "convert",
		Aliases: []string{"secretmappings", "sm", "secretmapping"},
		Short:   "Converts Secret resources in the path to ExternalSecret resources so they can be checked into git",
		Long:    labelLong,
		Example: fmt.Sprintf(labelExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	o.AddBaseFlags(cmd)
	o.AddSelectorFlags(cmd)

	cmd.Flags().StringVarP(&o.Dir, "dir", "d", ".", "the directory to look for the secret mapping files and version stream")
	cmd.Flags().StringVarP(&o.SourceDir, "source-dir", "", "", "the source directory to recursively look for the *.yaml or *.yml files to convert. If not specified defaults to 'config-root' in the dir")
	cmd.Flags().StringVarP(&o.VersionStreamDir, "version-stream-dir", "", "", "the directory containing the version stream. If not specified defaults to the 'versionStream' folder in the dir")
	cmd.Flags().StringVarP(&o.HelmSecretFolder, "helm-secrets-dir", "", "", "the directory where the helm secrets live with a folder per namespace and a file with a '.yaml' extension for each secret name. Defaults to $JX_HELM_SECRET_FOLDER")
	cmd.Flags().StringVarP(&o.DefaultNamespace, "default-namespace", "", "jx", "the default namespace if no namespace is specified in a Secret resource")

	cmd.AddCommand(cobras.SplitCommand(edit.NewCmdSecretMappingEdit()))
	return cmd, o
}

func (o *Options) Validate() error {
	err := o.BaseOptions.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to validate options")
	}
	o.Kinds = []string{"v1/Secret"}
	dir := o.Dir

	if o.SourceDir == "" {
		o.SourceDir = filepath.Join(o.Dir, "config-root")
	}
	if o.VersionStreamDir == "" {
		o.VersionStreamDir = filepath.Join(o.Dir, "versionStream")
	}

	if o.HelmSecretFolder == "" {
		o.HelmSecretFolder = extsecrets.DefaultHelmSecretFolder()
	}
	log.Logger().Debugf("moving converted Secret resources to the temporary helm secret folder %s", o.HelmSecretFolder)

	if o.SecretMapping == nil {
		var err error
		o.SecretMapping, _, err = secretmapping.LoadSecretMapping(dir, false)
		if err != nil {
			return errors.Wrapf(err, "failed to load secret mapping file")
		}
	}
	return nil
}

func (o *Options) Run() error {
	err := o.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to validate options")
	}

	modifyFn := func(node *yaml.RNode, path string) (bool, error) {
		var err error
		results, err := o.ModifyYAML(node, path)
		if err != nil {
			return results.Modified, err
		}
		namespace := results.Namespace
		name := results.Name
		helmDir := filepath.Join(o.HelmSecretFolder, namespace)
		err = os.MkdirAll(helmDir, files.DefaultDirWritePermissions)
		if err != nil {
			return false, errors.Wrapf(err, "failed to create helm secret dir %s", helmDir)
		}
		dest := filepath.Join(helmDir, name+".yaml")
		err = files.CopyFile(path, dest)
		if err != nil {
			return false, errors.Wrapf(err, "failed to copy secret %s to %s", path, dest)
		}
		return true, nil
	}

	err = kyamls.ModifyFiles(o.SourceDir, modifyFn, o.Filter)
	if err != nil {
		return errors.Wrapf(err, "failed to modify files")
	}
	return nil
}

// ModifyResults returns the results of modifying a file
type ModifyResults struct {
	Namespace string
	Name      string
	Modified  bool
}

// ModifyYAML modifies the given YAML files
func (o *Options) ModifyYAML(node *yaml.RNode, path string) (ModifyResults, error) {
	results := ModifyResults{}
	namespace := kyamls.GetNamespace(node, path)
	name := kyamls.GetName(node, path)

	hasData, err := hasSecretData(node, path)
	if err != nil {
		return results, errors.Wrapf(err, "failed to check if file has Secret data %s", path)
	}
	if !hasData {
		log.Logger().Debugf("not converting Secret %s in namespace %s to an ExternalSecret as it has no data", info(name), info(namespace))
		return results, nil
	}

	// An ExternalSecret must have at least one data or dataFrom entry — ESO's
	// webhook rejects it otherwise — so a Secret whose every key is unsecured
	// has nothing to fetch and stays a plain Secret.
	allUnsecured, err := o.allSecretDataUnsecured(node, path, name)
	if err != nil {
		return results, errors.Wrapf(err, "failed to check unsecured keys for %s", path)
	}
	if allUnsecured {
		log.Logger().Debugf("not converting Secret %s in namespace %s to an ExternalSecret as all of its keys are unsecured", info(name), info(namespace))
		return results, nil
	}

	secret := o.SecretMapping.FindRule(namespace, name)
	err = kyamls.SetStringValue(node, path, extsecrets.APIVersion, "apiVersion")
	if err != nil {
		return results, err
	}
	err = kyamls.SetStringValue(node, path, "ExternalSecret", "kind")
	if err != nil {
		return results, err
	}

	if secret.BackendType == "" {
		secret.BackendType = o.SecretMapping.Spec.BackendType
	}

	// The referenced store holds the backend-specific config (backendType,
	// roleArn, region, projectId, keyVaultName, vaultMountPoint, vaultRole), so
	// none of it is written onto the ExternalSecret. jx-secret reads the store
	// back at populate/edit time to recover it.
	err = kyamls.SetStringValue(node, path, extsecrets.DefaultSecretStoreName, "spec", "secretStoreRef", "name")
	if err != nil {
		return results, err
	}
	err = kyamls.SetStringValue(node, path, extsecrets.DefaultSecretStoreKind, "spec", "secretStoreRef", "kind")
	if err != nil {
		return results, err
	}

	// Validates the mapping and sets o.Prefix for GSM key naming; no spec fields
	// are written here.
	switch secret.BackendType {
	case v1alpha1.BackendTypeAlicloud:
		o.warnAlicloudUnsupported()

	case v1alpha1.BackendTypeGSM:
		if secret.GcpSecretsManager == nil {
			secret.GcpSecretsManager = &v1alpha1.GcpSecretsManager{}
		}
		if secret.GcpSecretsManager.ProjectID == "" &&
			o.SecretMapping.Spec.GcpSecretsManager.ProjectID == "" {
			return results, errors.New("missing secret mapping secret.GcpSecretsManager.ProjectID")
		}
		if secret.GcpSecretsManager.UniquePrefix != "" {
			o.Prefix = secret.GcpSecretsManager.UniquePrefix
		} else if o.SecretMapping.Spec.GcpSecretsManager.UniquePrefix != "" {
			o.Prefix = o.SecretMapping.Spec.GcpSecretsManager.UniquePrefix
		}

	case v1alpha1.BackendTypeAzure:
		if secret.AzureKeyVaultConfig == nil {
			secret.AzureKeyVaultConfig = &v1alpha1.AzureKeyVaultConfig{}
		}
		if secret.AzureKeyVaultConfig.KeyVaultName == "" &&
			(o.SecretMapping.Spec.AzureKeyVaultConfig == nil || o.SecretMapping.Spec.AzureKeyVaultConfig.KeyVaultName == "") {
			return results, errors.New("missing secret mapping secret.AzureKeyVaultConfig.KeyVaultName")
		}

	case v1alpha1.BackendTypeAWSSecretsManager:
		if secret.AwsSecretsManager == nil {
			secret.AwsSecretsManager = &v1alpha1.AwsSecretsManager{}
		}
		if secret.AwsSecretsManager.Region == "" && secret.Region == "" &&
			o.SecretMapping.Spec.AwsSecretsManager.Region == "" && o.SecretMapping.Spec.Region == "" {
			return results, errors.New("missing secret mapping secret.AwsSecretsManager.Region")
		}
	}

	err = o.convertData(node, path, secret.BackendType)
	if err != nil {
		return results, err
	}
	flag, err := o.moveMetadataToTemplate(node, path)
	if err != nil {
		return results, err
	}

	if namespace == "" {
		namespace = secret.Namespace
		if namespace == "" {
			namespace = o.DefaultNamespace
		}
	}
	results.Namespace = namespace
	results.Name = name
	results.Modified = flag
	return results, nil
}

// warnAlicloudUnsupported reports the backend as unusable once per run rather
// than once per Secret, so a repo full of them stays readable.
func (o *Options) warnAlicloudUnsupported() {
	if o.warnedAlicloud {
		return
	}
	o.warnedAlicloud = true
	log.Logger().Warnf("backendType %s is no longer supported: the External Secrets Operator has no alibaba provider in %s, so the generated ExternalSecrets cannot be resolved. Migrate these secrets to another backend.",
		v1alpha1.BackendTypeAlicloud, extsecrets.APIVersion)
}

// allSecretDataUnsecured reports whether every data and stringData key of the
// Secret is listed as unsecured in the mapping.
func (o *Options) allSecretDataUnsecured(node *yaml.RNode, path, secretName string) (bool, error) {
	if o.SecretMapping == nil {
		return false, nil
	}
	for _, dataPath := range []string{"data", "stringData"} {
		data, err := node.Pipe(yaml.Lookup(dataPath))
		if err != nil {
			return false, errors.Wrapf(err, "failed to get data for path %s", path)
		}
		if data == nil {
			continue
		}
		fields, err := data.Fields()
		if err != nil {
			return false, errors.Wrapf(err, "failed to find data fields for path %s", path)
		}
		for _, field := range fields {
			if !o.SecretMapping.IsSecretKeyUnsecured(secretName, field) {
				return false, nil
			}
		}
	}
	return true, nil
}

// hasSecretData returns true if the node has secret data fields
func hasSecretData(node *yaml.RNode, path string) (bool, error) {
	for _, dataPath := range []string{"data", "stringData"} {
		data, err := node.Pipe(yaml.Lookup(dataPath))
		if err != nil {
			return false, errors.Wrapf(err, "failed to get data for path %s", path)
		}

		var fields []string
		if data != nil {
			fields, err = data.Fields()
			if err != nil {
				return false, errors.Wrapf(err, "failed to find data fields for path %s", path)
			}
			if len(fields) > 0 {
				return true, nil
			}
		}
	}
	return false, nil
}

func (o *Options) convertData(node *yaml.RNode, path string, backendType v1alpha1.BackendType) error {
	secretName := kyamls.GetStringField(node, path, "metadata", "name")

	var contents []*yaml.Node
	style := node.Document().Style

	// Both data and stringData literals collect here: the target template has a
	// single data map, with no stringData equivalent.
	templateData := &yaml.Node{
		Kind:  yaml.MappingNode,
		Style: style,
	}
	rTemplateData := yaml.NewRNode(templateData)

	for _, dataPath := range []string{"data", "stringData"} {
		data, err := node.Pipe(yaml.Lookup(dataPath))
		if err != nil {
			return errors.Wrapf(err, "failed to get data for path %s", path)
		}

		var fields []string
		if data != nil {
			fields, err = data.Fields()
			if err != nil {
				return errors.Wrapf(err, "failed to find data fields for path %s", path)
			}
			complexSecretType := len(fields) > 1

			for _, field := range fields {

				if o.SecretMapping.IsSecretKeyUnsecured(secretName, field) {
					secretValue := kyamls.GetStringField(data, "", field)

					err = kyamls.SetStringValue(rTemplateData, path, secretValue, field)
					if err != nil {
						return errors.Wrapf(err, "failed to set string value for secret %s and key %s", secretName, field)
					}
					continue
				}

				newNode := &yaml.Node{
					Kind:  yaml.MappingNode,
					Style: style,
				}

				rNode := yaml.NewRNode(newNode)

				switch backendType {
				case v1alpha1.BackendTypeVault:
					err = o.modifyVault(node, rNode, field, secretName, path)

				case v1alpha1.BackendTypeGSM:
					err = o.modifyGSM(rNode, field, secretName, path)

				case v1alpha1.BackendTypeLocal:
					err = o.modifyLocal(rNode, field, secretName, path)

				case v1alpha1.BackendTypeAWSSecretsManager:
					err = o.modifyASM(rNode, field, secretName, path)

				default:
					err = o.modifyDefault(rNode, field, secretName, path, complexSecretType, false)
				}

				if err != nil {
					return errors.Wrapf(err, "failed to modify ExternalSecret with configuration")
				}
				contents = append(contents, newNode)
			}
		}
		err = node.PipeE(yaml.Clear(dataPath))
		if err != nil {
			return errors.Wrapf(err, "failed to remove %s", dataPath)
		}
	}

	if len(templateData.Content) != 0 {
		templateDataNode, err := node.Pipe(yaml.LookupCreate(yaml.MappingNode, "spec", "target", "template", "data"))
		if err != nil {
			return errors.Wrapf(err, "failed to lookup/create template data for path %s", path)
		}
		templateDataNode.SetYNode(&yaml.Node{
			Kind:    yaml.MappingNode,
			Content: templateData.Content,
			Style:   style,
		})

		// Without this the template's mergePolicy defaults to Replace, and the
		// Secret would contain only these unsecured literals — every key fetched
		// via spec.data would be dropped.
		err = kyamls.SetStringValue(node, path, string(esv1.MergePolicyMerge), "spec", "target", "template", "mergePolicy")
		if err != nil {
			return errors.Wrapf(err, "failed to set template mergePolicy for path %s", path)
		}
	}

	data, err := node.Pipe(yaml.LookupCreate(yaml.SequenceNode, "spec", "data"))
	if err != nil {
		return errors.Wrapf(err, "failed to replace data for path %s", path)
	}
	data.SetYNode(&yaml.Node{
		Kind:    yaml.SequenceNode,
		Content: contents,
		Style:   style,
	})

	return nil
}

// setRemoteRef writes a spec.data entry: secretKey plus a nested remoteRef of
// key, property and any extra fields.
func setRemoteRef(rNode *yaml.RNode, path, secretKey, key, property string, extra map[string]string) error {
	if err := kyamls.SetStringValue(rNode, path, secretKey, "secretKey"); err != nil {
		return err
	}
	if err := kyamls.SetStringValue(rNode, path, key, "remoteRef", "key"); err != nil {
		return err
	}
	if property != "" {
		if err := kyamls.SetStringValue(rNode, path, property, "remoteRef", "property"); err != nil {
			return err
		}
	}
	// sorted so the generated YAML is stable across runs
	fields := make([]string, 0, len(extra))
	for field := range extra {
		if extra[field] != "" {
			fields = append(fields, field)
		}
	}
	sort.Strings(fields)
	for _, field := range fields {
		if err := setQuotedStringValue(rNode, path, extra[field], "remoteRef", field); err != nil {
			return err
		}
	}
	return nil
}

// setQuotedStringValue sets a string field, quoting it when the plain form would
// be read back as a number or boolean. remoteRef.version is typed as a string in
// the ESO schema, so a bare `version: 1` is rejected by the CRD.
func setQuotedStringValue(rNode *yaml.RNode, path, value string, fields ...string) error {
	if err := kyamls.SetStringValue(rNode, path, value, fields...); err != nil {
		return err
	}
	if !needsQuoting(value) {
		return nil
	}
	field, err := rNode.Pipe(yaml.Lookup(fields...))
	if err != nil {
		return errors.Wrapf(err, "failed to look up %s at path %s", kyamls.JSONPath(fields...), path)
	}
	if field != nil {
		field.YNode().Style = yaml.SingleQuotedStyle
	}
	return nil
}

func needsQuoting(value string) bool {
	if value == "" {
		return false
	}
	var probe interface{}
	if err := yaml.Unmarshal([]byte(value), &probe); err != nil {
		return true
	}
	_, isString := probe.(string)
	return !isString
}

func (o *Options) modifyVault(node, rNode *yaml.RNode, field, secretName, path string) error {
	prefix := kyamls.GetStringField(node, path, "metadata", "annotations", "secret.jenkins-x.io/prefix")
	if prefix != "" {
		prefix += "/"
	}
	// trim the suffix from the name and use it on the property?
	property := field
	secretPath := strings.ReplaceAll(secretName, "-", "/")
	names := strings.Split(secretPath, "/")
	if len(names) > 1 && names[len(names)-1] == property {
		secretPath = strings.Join(names[0:len(names)-1], "/")
	}
	// The vault provider takes the bare path; the KV mount is configured on the
	// ClusterSecretStore rather than prefixed here.
	key := prefix + secretPath

	if o.SecretMapping != nil {
		mapping := o.SecretMapping.Find(secretName, field)
		if mapping != nil {
			if mapping.Key != "" {
				key = mapping.Key
			}
			if mapping.Property != "" {
				property = mapping.Property
			}
		}
	}

	return setRemoteRef(rNode, path, field, key, property, nil)
}

func (o *Options) modifyDefault(rNode *yaml.RNode, field, secretName, path string, complexType, supportsVersionStage bool) error {
	var key string
	property := ""
	if complexType {
		property = field
	}

	if o.Prefix != "" {
		key = o.Prefix + "-" + secretName
	} else {
		key = secretName
	}

	isBinary := false
	versionStage := ""

	if o.SecretMapping != nil {
		mapping := o.SecretMapping.Find(secretName, field)
		if mapping != nil {
			if mapping.Key != "" {
				key = mapping.Key
			}
			if mapping.Property != "" {
				property = mapping.Property
			}
			if mapping.VersionStage != "" {
				versionStage = mapping.VersionStage
			}
			isBinary = mapping.IsBinary
		}
		if versionStage == "" {
			versionStage = o.SecretMapping.Spec.VersionStage
		}
	}

	if key == "" {
		return fmt.Errorf("no key found when mapping secret %s", secretName)
	}

	// ESO has no separate versionStage field. AWS Secrets Manager reads
	// remoteRef.version as a version stage, so the mapping's value carries over
	// there. Other back ends treat it as a concrete version id, where a stage
	// name would not resolve, so it is dropped rather than mistranslated.
	extra := map[string]string{}
	if supportsVersionStage {
		extra["version"] = versionStage
	} else if versionStage != "" {
		log.Logger().Debugf("ignoring versionStage %s for secret %s: only AWS Secrets Manager supports version stages", versionStage, secretName)
	}
	if isBinary {
		extra["decodingStrategy"] = string(esv1.ExternalSecretDecodeBase64)
	}
	return setRemoteRef(rNode, path, field, key, property, extra)
}

func (o *Options) modifyLocal(rNode *yaml.RNode, field, secretName, path string) error {
	key := field
	property := field
	if o.SecretMapping != nil {
		mapping := o.SecretMapping.Find(secretName, field)
		if mapping != nil {
			if mapping.Key != "" {
				key = mapping.Key
			}
			if mapping.Property != "" {
				property = mapping.Property
			}
		}
	}

	return setRemoteRef(rNode, path, field, key, property, nil)
}

func (o *Options) modifyGSM(rNode *yaml.RNode, field, secretName, path string) error {
	var property string
	var key string

	if o.Prefix != "" {
		key = o.Prefix + "-" + secretName
	} else {
		key = secretName
	}

	version := "latest"
	if o.SecretMapping != nil {
		mapping := o.SecretMapping.Find(secretName, field)
		if mapping != nil {
			if mapping.Key != "" {
				if o.Prefix != "" {
					key = o.Prefix + "-" + mapping.Key
				} else {
					key = mapping.Key
				}
			}
			if mapping.Property != "" {
				property = mapping.Property
			}
		}
		secret := o.SecretMapping.FindSecret(secretName)
		if secret != nil && secret.GcpSecretsManager != nil {
			if secret.GcpSecretsManager.Version != "" {
				version = secret.GcpSecretsManager.Version
			}
		}
	}

	key = strings.ToLower(key)

	if key == "" {
		return fmt.Errorf("no key found when mapping secret %s", secretName)
	}

	if property == "" {
		property = field
	}
	return setRemoteRef(rNode, path, field, key, property, map[string]string{"version": version})
}

func (o *Options) modifyASM(rNode *yaml.RNode, field, secretName, path string) error {
	return o.modifyDefault(rNode, field, secretName, path, true, true)
}

func (o *Options) moveMetadataToTemplate(node *yaml.RNode, path string) (bool, error) {
	// Moving type/labels/annotations under the target template is what makes the
	// operator stamp them onto the Secret it creates.
	typeValue := kyamls.GetStringField(node, path, "type")

	labels, err := node.Pipe(yaml.Lookup("metadata", "labels"))
	if err != nil {
		return false, errors.Wrapf(err, "failed to get labels")
	}
	annotations, err := node.Pipe(yaml.Lookup("metadata", "annotations"))
	if err != nil {
		return false, errors.Wrapf(err, "failed to get annotations")
	}

	if typeValue != "" || labels != nil || annotations != nil {
		var templateNode *yaml.RNode
		templateNode, err = node.Pipe(yaml.LookupCreate(yaml.MappingNode, "spec", "target", "template"))
		if err != nil {
			return false, errors.Wrapf(err, "failed to create spec.target.template")
		}
		if templateNode == nil {
			return false, errors.Errorf("could not create spec.target.template")
		}

		if annotations != nil {
			var newAnnotations *yaml.RNode
			newAnnotations, err = templateNode.Pipe(yaml.LookupCreate(yaml.MappingNode, "metadata", "annotations"))
			if err != nil {
				return false, errors.Wrapf(err, "failed to set annotations on template")
			}
			newAnnotations.SetYNode(annotations.YNode())
		}
		if labels != nil {
			var newLabels *yaml.RNode
			newLabels, err = templateNode.Pipe(yaml.LookupCreate(yaml.MappingNode, "metadata", "labels"))
			if err != nil {
				return false, errors.Wrapf(err, "failed to set labels on template")
			}
			newLabels.SetYNode(labels.YNode())
		}
		if typeValue != "" {
			err = kyamls.SetStringValue(templateNode, path, typeValue, "type")
			if err != nil {
				return false, errors.Wrapf(err, "failed to set type on template")
			}
		}
		err = node.PipeE(yaml.Clear("type"))
		if err != nil {
			return false, errors.Wrapf(err, "failed to clear type")
		}
		var metadata *yaml.RNode
		metadata, err = node.Pipe(yaml.Lookup("metadata"))
		if err != nil {
			return false, errors.Wrapf(err, "failed to get metadata")
		}
		if metadata != nil {
			err = metadata.PipeE(yaml.Clear("annotations"))
			if err != nil {
				return false, errors.Wrapf(err, "failed to clear metadata annotations")
			}
		}
	}

	// add the optional schema annotation if we can find the schema
	schemaAnnotation, err := o.findSchemaObjectAnnotation(node, path)
	if err != nil {
		return false, errors.Wrapf(err, "failed to find schema for secret at path %s", path)
	}
	if schemaAnnotation != "" {
		err = node.PipeE(yaml.SetAnnotation(extsecrets.SchemaObjectAnnotation, schemaAnnotation))
		if err != nil {
			return false, errors.Wrapf(err, "failed to add mandatory annotation to file %s", path)
		}

		templateAnnotationsNode, err := node.Pipe(yaml.LookupCreate(yaml.MappingNode, "spec", "target", "template", "metadata", "annotations"))
		if err != nil {
			return false, errors.Wrapf(err, "failed to create the template annotations node")
		}
		err = templateAnnotationsNode.PipeE(yaml.SetField(extsecrets.SchemaObjectAnnotation, yaml.NewScalarRNode(schemaAnnotation)))
		if err != nil {
			return false, errors.Wrapf(err, "failed to add template annotation to file %s", path)
		}
	}
	return true, nil
}

func (o *Options) findSchemaSchemaObject(node *yaml.RNode, path string) (*schema.Object, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find absolute path for %s", path)
	}
	paths := strings.Split(absPath, string(os.PathSeparator))
	if len(paths) < 2 {
		log.Logger().Warnf("cannot find the chart name from such a small path %s", absPath)
		return nil, nil
	}
	lastDir := paths[len(paths)-2]
	g := filepath.Join(o.VersionStreamDir, "charts", "*", lastDir, "secret-schema.yaml")
	fileSlice, err := filepath.Glob(g)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find files with glob: %s", g)
	}

	// lets also look in the charts dir for secret schema files
	chartsDir := filepath.Join(o.Dir, "charts")
	exists, err := files.DirExists(chartsDir)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to detect dir exists %s", chartsDir)
	}
	if exists {
		g = filepath.Join(chartsDir, "*", lastDir, "secret-schema.yaml")
		fileSlice2, err := filepath.Glob(g)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to find files with glob: %s", g)
		}
		if len(fileSlice2) > 0 {
			fileSlice = append(fileSlice, fileSlice2...)
		}
	}

	if len(fileSlice) == 0 {
		return nil, nil
	}
	name := kyamls.GetName(node, path)
	return schemas.LoadSchemaObjectFromFiles(name, fileSlice)
}

func (o *Options) findSchemaObjectAnnotation(node *yaml.RNode, path string) (string, error) {
	sch, err := o.findSchemaSchemaObject(node, path)
	if sch == nil || err != nil {
		return "", err
	}
	// lets convert to YAML so we can store it as an annotation
	text, err := schemas.ToAnnotationString(sch)
	if err != nil {
		return "", errors.Wrapf(err, "failed to convert schema for path %s to YAML", path)
	}
	return text, nil
}
