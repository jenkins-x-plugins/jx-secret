package replicate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	jxcore "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"
	jenkinsv1 "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kyamls"
	"github.com/jenkins-x/jx-helpers/v3/pkg/options"
	"github.com/jenkins-x/jx-helpers/v3/pkg/stringhelpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yamls"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets"
	"github.com/jenkins-x/jx-secret/pkg/rootcmd"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var (
	labelLong = templates.LongDesc(`
		Replicates the given ExternalSecret resources into other Environments or Namespaces
`)

	labelExample = templates.Examples(`
		# replicates the labeled ExternalSecret resources to the local permanent Environment namespaces (e.g. Staging and Production)
		%s replicate --label secret.jenkins-x.io/replica-source=true

		# replicates the ExternalSecret resources to the local Environments
		%s replicate --name=mysecretname --to jx-staging,jx-production
	`)
)

// LabelOptions the options for the command
type Options struct {
	File          string
	Dir           string
	OutputDir     string
	NamespacesDir string
	From          string
	Selector      string
	Name          []string
	To            []string
}

// NewCmdReplicate creates a command object for the command
func NewCmdReplicate() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "replicate",
		Short:   "Replicates the given ExternalSecret resources into other Environments or Namespaces",
		Long:    labelLong,
		Example: fmt.Sprintf(labelExample, rootcmd.BinaryName, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.File, "file", "f", "t", "the ExternalSecret to replicate")
	cmd.Flags().StringVarP(&o.Selector, "selector", "s", "", "defines the label selector to find the ExternalSecret resources to replicate")
	cmd.Flags().StringArrayVarP(&o.Name, "name", "n", nil, "specifies the names of the ExternalSecrets to replicate if not using a selector")
	cmd.Flags().StringVarP(&o.From, "from", "", "", "one or more Namespaces to replicate the ExternalSecret to")
	cmd.Flags().StringArrayVarP(&o.To, "to", "t", nil, "one or more Namespaces to replicate the ExternalSecret to")
	cmd.Flags().StringVarP(&o.OutputDir, "output-dir", "o", "", "the output directory which defaults to 'config-root' in the directory")
	return cmd, o
}

func (o *Options) Run() error {
	path := o.File
	if path == "" {
		return options.MissingOption("file")
	}
	if len(o.Name) == 0 && o.Selector == "" {
		return options.MissingOption("name")
	}
	if o.From == "" {
		// lets default the namespace
		o.From = jxcore.DefaultNamespace
		if o.From == "" {
			return options.MissingOption("from")
		}
	}
	if o.OutputDir == "" {
		o.OutputDir = filepath.Join(o.Dir, "config-root")
	}
	if o.NamespacesDir == "" {
		o.NamespacesDir = filepath.Join(o.OutputDir, "namespaces")
	}
	dir := filepath.Join(o.NamespacesDir, o.From)

	if len(o.To) == 0 {
		err := o.discoverEnvironmentNamespaces(dir)
		if err != nil {
			return errors.Wrapf(err, "failed to discover the Environment namespaces in dir %s", dir)
		}

		if len(o.To) == 0 {
			log.Logger().Warnf("no --to specified and no remote Environments found")
			return nil
		}
	}

	found := map[string]bool{}

	filter := kyamls.Filter{
		Kinds: []string{"ExternalSecret"},
		Names: o.Name,
	}
	if o.Selector != "" {
		selector, err := metav1.ParseToLabelSelector(o.Selector)
		if err != nil {
			return errors.Wrapf(err, "failed to parse selector %s", o.Selector)
		}
		filter.Selector = selector.MatchLabels
	}
	modifyFn := func(node *yaml.RNode, path string) (bool, error) {
		name := kyamls.GetName(node, path)
		found[name] = true

		for _, ns := range o.To {
			relPath, err := filepath.Rel(dir, path)
			if err != nil {
				return false, errors.Wrapf(err, "failed to get relative path of %s from %s", path, dir)
			}

			err = node.PipeE(yaml.LookupCreate(yaml.ScalarNode, "metadata", "namespace"), yaml.FieldSetter{StringValue: ns})
			if err != nil {
				return false, errors.Wrapf(err, "failed to set metadata.namespace to %s", ns)
			}

			err = node.PipeE(yaml.SetAnnotation(extsecrets.ReplicaAnnotation, "true"))
			if err != nil {
				return false, errors.Wrapf(err, "failed to add replica annotation for path %s", path)
			}

			outFile := filepath.Join(o.NamespacesDir, ns, relPath)
			outDir := filepath.Dir(outFile)
			err = os.MkdirAll(outDir, files.DefaultDirWritePermissions)
			if err != nil {
				return false, errors.Wrapf(err, "failed to create output directory %s", outDir)
			}

			err = yaml.WriteFile(node, outFile)
			if err != nil {
				return false, errors.Wrapf(err, "failed to write ExternalSecret %s/%s to file %s", ns, name, outFile)
			}
			log.Logger().Infof("replicated ExternalSecret %s/%s to %s", ns, name, outFile)
		}
		err := o.addReplicatedLocalBackendAnnotation(path)
		if err != nil {
			return false, errors.Wrapf(err, "failed to annotate replicated local backend")
		}
		return false, nil
	}

	err := kyamls.ModifyFiles(dir, modifyFn, filter)
	if err != nil {
		return errors.Wrapf(err, "failed to replicate secrets in namespace %s in dir %s", o.From, dir)
	}

	for _, name := range o.Name {
		if !found[name] {
			log.Logger().Warnf("could not find ExternalSecret %s in namespace %s", name, o.From)
		}
	}
	return nil
}

func (o *Options) addReplicatedLocalBackendAnnotation(path string) error {
	node, err := yaml.ReadFile(path)
	if err != nil {
		return errors.Wrapf(err, "failed to load %s", path)
	}

	backend, err := node.Pipe(yaml.Lookup("spec", "backendType"))
	if err != nil {
		return errors.Wrapf(err, "failed to find backendType for %s", path)
	}
	if backend == nil {
		return nil
	}

	backendType, err := backend.String()
	if err != nil {
		return errors.Wrapf(err, "failed to get backendType for %s", path)
	}
	backendType = strings.TrimSpace(backendType)
	if backendType != "local" {
		log.Logger().Infof("ignoring backend type %s", backendType)
		return nil
	}

	// lets add an annotation
	err = node.PipeE(yaml.SetAnnotation(extsecrets.ReplicateToAnnotation, strings.Join(o.To, ",")))
	if err != nil {
		return errors.Wrapf(err, "failed to add replicate annotation for path %s", path)
	}
	err = yaml.WriteFile(node, path)
	if err != nil {
		return errors.Wrapf(err, "failed to save file %s", path)
	}
	return nil
}

func (o *Options) discoverEnvironmentNamespaces(dir string) error {
	// lets try find the environment namespaces by default
	modifyFn := func(node *yaml.RNode, path string) (bool, error) {
		env := &jenkinsv1.Environment{}
		err := yamls.LoadFile(path, env)
		if err != nil {
			return false, errors.Wrapf(err, "failed to parse Environment: %s", path)
		}
		if env.Spec.Kind != jenkinsv1.EnvironmentKindTypePermanent {
			return false, nil
		}

		ens := env.Spec.Namespace
		if ens != "" && stringhelpers.StringArrayIndex(o.To, ens) < 0 {
			o.To = append(o.To, ens)
		}
		return false, nil
	}

	err := kyamls.ModifyFiles(dir, modifyFn, kyamls.Filter{
		Kinds: []string{"Environment"},
	})
	if err != nil {
		return errors.Wrapf(err, "failed to find Environment namespaces in dir %s", dir)
	}
	return nil
}
