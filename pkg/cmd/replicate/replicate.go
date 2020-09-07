package replicate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx-api/pkg/config"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/pkg/files"
	"github.com/jenkins-x/jx-helpers/pkg/kyamls"
	"github.com/jenkins-x/jx-helpers/pkg/options"
	"github.com/jenkins-x/jx-helpers/pkg/stringhelpers"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets"
	"github.com/jenkins-x/jx-secret/pkg/rootcmd"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var (
	labelLong = templates.LongDesc(`
		Replicates the given ExternalSecret resources into other Environments or Namespaces
`)

	labelExample = templates.Examples(`
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
		Example: fmt.Sprintf(labelExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.File, "file", "f", "t", "the ExternalSecret to replicate")
	cmd.Flags().StringVarP(&o.From, "from", "", "", "one or more Namespaces to replicate the ExternalSecret to")
	cmd.Flags().StringArrayVarP(&o.To, "to", "t", nil, "one or more Namespaces to replicate the ExternalSecret to")
	cmd.Flags().StringVarP(&o.OutputDir, "output-dir", "o", "", "the output directory which defaults to 'config-root' in the directory")
	cmd.Flags().StringArrayVarP(&o.Name, "name", "n", nil, "one or more names of ExternalSecrets to replicate")
	return cmd, o
}

func (o *Options) Run() error {
	path := o.File
	if path == "" {
		return options.MissingOption("file")
	}
	if len(o.To) == 0 {
		return options.MissingOption("to")
	}
	if len(o.Name) == 0 {
		return options.MissingOption("name")
	}
	if o.From == "" {
		// lets default the namespace from the cluster namespace
		requirements, _, err := config.LoadRequirementsConfig(o.Dir, false)
		if err != nil {
			return errors.Wrapf(err, "failed to load requirements in dir %s", o.Dir)
		}
		o.From = requirements.Cluster.Namespace
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

	found := map[string]bool{}

	filter := kyamls.Filter{
		Kinds: []string{"ExternalSecret"},
	}
	modifyFn := func(node *yaml.RNode, path string) (bool, error) {
		name := kyamls.GetName(node, path)
		if stringhelpers.StringArrayIndex(o.Name, name) < 0 {
			return false, nil
		}
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
		err := o.addReplciatedLocalBackendAnnotation(path)
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

func (o *Options) addReplciatedLocalBackendAnnotation(path string) error {
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
	err = node.PipeE(yaml.SetAnnotation(extsecrets.ReplicateAnnotation, strings.Join(o.To, ",")))
	if err != nil {
		return errors.Wrapf(err, "failed to add replicate annotation for path %s", path)
	}
	err = yaml.WriteFile(node, path)
	if err != nil {
		return errors.Wrapf(err, "failed to save file %s", path)
	}
	return nil
}
