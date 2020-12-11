package convert

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/jenkins-x/jx-secret/pkg/cmd/convert/edit"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets"
	"github.com/jenkins-x/jx-secret/pkg/schemas"
	"github.com/jenkins-x/jx-secret/pkg/vaults"

	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kyamls"
	"github.com/jenkins-x/jx-secret/pkg/apis/mapping/v1alpha1"
	schema "github.com/jenkins-x/jx-secret/pkg/apis/schema/v1alpha1"
	"github.com/jenkins-x/jx-secret/pkg/rootcmd"
	"github.com/jenkins-x/jx-secret/pkg/secretmapping"
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

	secretFilter = kyamls.Filter{
		Kinds: []string{"v1/Secret"},
	}
)

// LabelOptions the options for the command
type Options struct {
	Dir              string
	SourceDir        string
	VersionStreamDir string
	Backend          string
	VaultMountPoint  string
	VaultRole        string
	SecretMapping    *v1alpha1.SecretMapping

	Prefix string
}

// NewCmdSecretConvert creates a command object for the command
func NewCmdSecretConvert() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "convert",
		Aliases: []string{"secretmappings", "sm", "secretmapping"},
		Short:   "Converts all Secret resources in the path to ExternalSecret resources so they can be checked into git",
		Long:    labelLong,
		Example: fmt.Sprintf(labelExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Dir, "dir", "d", ".", "the directory to look for the secret mapping files and version stream")
	cmd.Flags().StringVarP(&o.SourceDir, "source-dir", "", "", "the source directory to recursively look for the *.yaml or *.yml files to convert. If not specified defaults to 'config-root' in the dir")
	cmd.Flags().StringVarP(&o.VersionStreamDir, "version-stream-dir", "", "", "the directory containing the version stream. If not specified defaults to the 'versionStream' folder in the dir")
	cmd.Flags().StringVarP(&o.VaultMountPoint, "vault-mount-point", "m", "kubernetes", "the vault authentication mount point")
	cmd.Flags().StringVarP(&o.VaultRole, "vault-role", "r", vaults.DefaultVaultNamespace, "the vault role that will be used to fetch the secrets. This role will need to be bound to kubernetes-external-secret's ServiceAccount; see Vault's documentation: https://www.vaultproject.io/docs/auth/kubernetes.html")

	cmd.AddCommand(cobras.SplitCommand(edit.NewCmdSecretMappingEdit()))
	return cmd, o
}

func (o *Options) Run() error {
	dir := o.Dir

	if o.SourceDir == "" {
		o.SourceDir = filepath.Join(o.Dir, "config-root")
	}
	if o.VersionStreamDir == "" {
		o.VersionStreamDir = filepath.Join(o.Dir, "versionStream")
	}

	if o.SecretMapping == nil {
		var err error
		o.SecretMapping, _, err = secretmapping.LoadSecretMapping(dir, false)
		if err != nil {
			return errors.Wrapf(err, "failed to load secret mapping file")
		}
	}

	modifyFn := func(node *yaml.RNode, path string) (bool, error) {
		namespace := kyamls.GetNamespace(node, path)
		name := kyamls.GetName(node, path)

		hasData, err := hasSecretData(node, path)
		if err != nil {
			return false, errors.Wrapf(err, "failed to check if file has Secret data %s", path)
		}
		if !hasData {
			log.Logger().Infof("not converting Secret %s in namespace %s to an ExternalSecret as it has no data", info(name), info(namespace))
			return false, nil
		}

		secret := o.SecretMapping.FindRule(namespace, name)
		err = kyamls.SetStringValue(node, path, "kubernetes-client.io/v1", "apiVersion")
		if err != nil {
			return false, err
		}
		err = kyamls.SetStringValue(node, path, "ExternalSecret", "kind")
		if err != nil {
			return false, err
		}

		if secret.BackendType == "" {
			secret.BackendType = o.SecretMapping.Spec.Defaults.BackendType
		}
		err = kyamls.SetStringValue(node, path, string(secret.BackendType), "spec", "backendType")
		if err != nil {
			return false, err
		}

		if secret.BackendType == v1alpha1.BackendTypeGSM {
			if secret.GcpSecretsManager == nil {
				secret.GcpSecretsManager = &v1alpha1.GcpSecretsManager{}
			}
			if secret.GcpSecretsManager.ProjectID != "" {
				err = kyamls.SetStringValue(node, path, secret.GcpSecretsManager.ProjectID, "spec", "projectId")
				if err != nil {
					return false, err
				}
			} else if o.SecretMapping.Spec.Defaults.GcpSecretsManager.ProjectID != "" {
				err = kyamls.SetStringValue(node, path, o.SecretMapping.Spec.Defaults.GcpSecretsManager.ProjectID, "spec", "projectId")
				if err != nil {
					return false, err
				}
			} else {
				return false, errors.New("missing secret mapping secret.GcpSecretsManager.ProjectID")
			}

			// if we have a unique prefix for the specific secret or a default one then set it to use as a gsm secret prefix later
			if secret.GcpSecretsManager.UniquePrefix != "" {
				o.Prefix = secret.GcpSecretsManager.UniquePrefix
			} else if o.SecretMapping.Spec.Defaults.GcpSecretsManager.UniquePrefix != "" {
				o.Prefix = o.SecretMapping.Spec.Defaults.GcpSecretsManager.UniquePrefix
			}
		}

		if secret.BackendType == v1alpha1.BackendTypeVault {
			err = kyamls.SetStringValue(node, path, o.VaultMountPoint, "spec", "vaultMountPoint")
			if err != nil {
				return false, err
			}
			err = kyamls.SetStringValue(node, path, o.VaultRole, "spec", "vaultRole")
			if err != nil {
				return false, err
			}
		}

		if secret.BackendType == v1alpha1.BackendTypeAzure {
			if secret.AzureKeyVaultConfig == nil {
				secret.AzureKeyVaultConfig = &v1alpha1.AzureKeyVaultConfig{}
			}
			if secret.AzureKeyVaultConfig.KeyVaultName != "" {
				err = kyamls.SetStringValue(node, path, secret.AzureKeyVaultConfig.KeyVaultName, "spec", "keyVaultName")
				if err != nil {
					return false, err
				}
			} else if o.SecretMapping.Spec.Defaults.AzureKeyVaultConfig != nil && o.SecretMapping.Spec.Defaults.AzureKeyVaultConfig.KeyVaultName != "" {
				err = kyamls.SetStringValue(node, path, o.SecretMapping.Spec.Defaults.AzureKeyVaultConfig.KeyVaultName, "spec", "keyVaultName")
				if err != nil {
					return false, err
				}
			} else {
				return false, errors.New("missing secret mapping secret.AzureKeyVaultConfig.KeyVaultName")
			}
			if err != nil {
				return false, err
			}
		}

		flag, err := o.convertData(node, path, secret.BackendType)
		if err != nil {
			return flag, err
		}
		flag, err = o.moveMetadataToTemplate(node, path)
		if err != nil {
			return flag, err
		}
		return true, nil
	}

	err := kyamls.ModifyFiles(o.SourceDir, modifyFn, secretFilter)
	if err != nil {
		return errors.Wrapf(err, "failed to modify files")
	}
	return nil
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

func (o *Options) convertData(node *yaml.RNode, path string, backendType v1alpha1.BackendType) (bool, error) {
	secretName := kyamls.GetStringField(node, path, "metadata", "name")

	var contents []*yaml.Node
	style := node.Document().Style

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
			for _, field := range fields {
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

				case v1alpha1.BackendTypeAzure:
					err = o.modifyAzure(rNode, field, secretName, path)

				}

				if err != nil {
					return false, errors.Wrapf(err, "failed to modify ExternalSecret with configuration")
				}
				contents = append(contents, newNode)
			}
		}
		err = node.PipeE(yaml.Clear(dataPath))
		if err != nil {
			return false, errors.Wrapf(err, "failed to remove %s", dataPath)
		}
	}

	data, err := node.Pipe(yaml.LookupCreate(yaml.SequenceNode, "spec", "data"))
	if err != nil {
		return false, errors.Wrapf(err, "failed to replace data for path %s", path)
	}
	if data == nil {
		return false, errors.Errorf("no data node for path %s", path)
	}
	data.SetYNode(&yaml.Node{
		Kind:    yaml.SequenceNode,
		Content: contents,
		Style:   style,
	})
	return true, nil
}

func (o *Options) modifyVault(node *yaml.RNode, rNode *yaml.RNode, field, secretName, path string) error {
	prefix := kyamls.GetStringField(node, path, "metadata", "annotations", "secret.jenkins-x.io/prefix")
	if prefix != "" {
		prefix = prefix + "/"
	} else {
		prefix = ""
	}
	// trim the suffix from the name and use it on the property?
	property := field
	secretPath := strings.ReplaceAll(secretName, "-", "/")
	names := strings.Split(secretPath, "/")
	if len(names) > 1 && names[len(names)-1] == property {
		secretPath = strings.Join(names[0:len(names)-1], "/")
	}
	key := "secret/data/" + prefix + secretPath

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

	err := kyamls.SetStringValue(rNode, path, field, "name")
	if err != nil {
		return err
	}
	err = kyamls.SetStringValue(rNode, path, key, "key")
	if err != nil {
		return err
	}
	err = kyamls.SetStringValue(rNode, path, property, "property")
	if err != nil {
		return err
	}
	return nil
}

func (o *Options) modifyAzure(rNode *yaml.RNode, field, secretName, path string) error {

	var property string
	var key string

	if o.Prefix != "" {
		key = o.Prefix + "-" + secretName
	} else {
		key = secretName
	}

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

	if key == "" {
		return fmt.Errorf("no key found when mapping secret %s", secretName)
	}

	err := kyamls.SetStringValue(rNode, path, field, "name")
	if err != nil {
		return err
	}
	err = kyamls.SetStringValue(rNode, path, key, "key")
	if err != nil {
		return err
	}
	err = kyamls.SetStringValue(rNode, path, property, "property")
	if err != nil {
		return err
	}
	return nil
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

	err := kyamls.SetStringValue(rNode, path, field, "name")
	if err != nil {
		return err
	}
	err = kyamls.SetStringValue(rNode, path, key, "key")
	if err != nil {
		return err
	}
	err = kyamls.SetStringValue(rNode, path, property, "property")
	if err != nil {
		return err
	}
	return nil
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

	err := kyamls.SetStringValue(rNode, path, field, "name")
	if err != nil {
		return err
	}
	err = kyamls.SetStringValue(rNode, path, key, "key")
	if err != nil {
		return err
	}
	if property != "" {
		err = kyamls.SetStringValue(rNode, path, property, "property")
		if err != nil {
			return err
		}
	}
	err = kyamls.SetStringValue(rNode, path, version, "version")
	if err != nil {
		return err
	}
	return nil
}

func (o *Options) moveMetadataToTemplate(node *yaml.RNode, path string) (bool, error) {
	// lets move annotations/labels/type  over to the template field
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
		templateNode, err = node.Pipe(yaml.LookupCreate(yaml.MappingNode, "spec", "template"))
		if err != nil {
			return false, errors.Wrapf(err, "failed to set kind")
		}
		if templateNode == nil {
			return false, errors.Errorf("could not create spec.template")
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
				return false, errors.Wrapf(err, "failed to set annotations on template")
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

		templateAnnotationsNode, err := node.Pipe(yaml.LookupCreate(yaml.MappingNode, "spec", "template", "metadata", "annotations"))
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
