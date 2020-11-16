package importcmd

import (
	"fmt"
	"io/ioutil"

	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube"
	"github.com/jenkins-x/jx-helpers/v3/pkg/options"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	v1 "github.com/jenkins-x/jx-secret/pkg/apis/external/v1"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets/editor"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets/editor/factory"
	"github.com/jenkins-x/jx-secret/pkg/rootcmd"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

var (
	editLong = templates.LongDesc(`
		Imports a YAML of secret values into the underlying secret store
`)

	editExample = templates.Examples(`
		%s import -f mysecrets.yaml
	`)
)

// Options the options for the command
type Options struct {
	File               string
	Namespace          string
	SecretClient       extsecrets.Interface
	KubeClient         kubernetes.Interface
	CommandRunner      cmdrunner.CommandRunner
	QuietCommandRunner cmdrunner.CommandRunner
	FailOnUnknownKey   bool
	ExternalSecrets    []*v1.ExternalSecret
	Handlers           map[string]*backendHandler

	// EditorCache the optional cache of editors
	EditorCache map[string]editor.Interface
}

// NewCmdImport creates a command object for the command
func NewCmdImport() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "import",
		Short:   "Imports a YAML file of secret values",
		Long:    editLong,
		Example: fmt.Sprintf(editExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.File, "file", "f", "", "the name of the file to import")
	cmd.Flags().StringVarP(&o.Namespace, "ns", "n", "", "the namespace to filter the ExternalSecret resources")
	cmd.Flags().BoolVarP(&o.FailOnUnknownKey, "fail-on-unknown-key", "", false, "should the command fail if a key from the YAML file is unknown")
	return cmd, o
}

// Run implements the command
func (o *Options) Run() error {
	if o.EditorCache == nil {
		o.EditorCache = map[string]editor.Interface{}
	}
	fileName := o.File
	if fileName == "" {
		return options.MissingOption("file")
	}

	m := map[string]interface{}{}
	exists, err := files.FileExists(fileName)
	if err != nil {
		return errors.Wrapf(err, "failed to check if file exists %s", fileName)
	}
	if !exists {
		return errors.Errorf("file does not exist: %s", fileName)
	}
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		return errors.Wrapf(err, "failed to load file %s", fileName)
	}
	err = yaml.Unmarshal(data, &m)
	if err != nil {
		return errors.Wrapf(err, "failed to parse YAML file %s", fileName)
	}

	if o.SecretClient == nil {
		o.SecretClient, err = extsecrets.NewClient(nil)
		if err != nil {
			return errors.Wrapf(err, "failed to create extsecrets client")
		}
	}
	o.KubeClient, o.Namespace, err = kube.LazyCreateKubeClientAndNamespace(o.KubeClient, o.Namespace)
	if err != nil {
		return errors.Wrapf(err, "failed to create kube Client")
	}

	resources, err := o.SecretClient.List(o.Namespace, metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to find external secrets")
	}
	o.ExternalSecrets = resources

	log.Logger().Debugf("found %d ExternalSecret resources", len(resources))

	for _, r := range resources {
		name := r.Name
		backendType := r.Spec.BackendType

		if o.Handlers == nil {
			o.Handlers = map[string]*backendHandler{}
		}
		for _, data := range r.Spec.Data {
			key := extsecrets.SimplifyKey(backendType, data.Key)
			property := data.Property
			handler := o.Handlers[key]
			if handler == nil {
				var e editor.Interface
				e, err = factory.NewEditor(o.EditorCache, r, o.CommandRunner, o.QuietCommandRunner, o.KubeClient)
				if err != nil {
					return errors.Wrapf(err, "failed to create e for secret %s of type %s", name, backendType)
				}
				handler = &backendHandler{
					Properties: map[string][]string{},
					Editor:     e,
				}
				o.Handlers[key] = handler
			}
			handler.Properties[key] = append(handler.Properties[key], property)
		}
	}

	err = o.updateProperties(m, "")
	if err != nil {
		return errors.Wrapf(err, "failed to import properties")
	}
	return nil
}

func (o *Options) updateProperties(m map[string]interface{}, path string) error {
	keyProperties := &editor.KeyProperties{
		Key: path,
	}
	for k, v := range m {
		childPath := k
		if path != "" {
			childPath = path + "/" + k
		}
		childMap, ok := v.(map[string]interface{})
		if ok {
			err := o.updateProperties(childMap, childPath)
			if err != nil {
				return errors.Wrapf(err, "failed to import value %#v to path %s", v, childPath)
			}
			continue
		}
		value, ok := v.(string)
		if !ok {
			return errors.Errorf("could not handle non string value %#v for path %s", v, childPath)
		}
		keyProperties.Properties = append(keyProperties.Properties, editor.PropertyValue{
			Property: k,
			Value:    value,
		})
	}
	if len(keyProperties.Properties) == 0 {
		return nil
	}

	backendHandler := o.Handlers[path]
	if backendHandler == nil {
		if o.FailOnUnknownKey {
			return errors.Errorf("could not find backend handler for path %s", path)
		}
		log.Logger().Warnf("the path %s does not map to an ExternalSecret key", path)
		return nil
	}

	err := backendHandler.Editor.Write(keyProperties)
	if err != nil {
		return errors.Wrapf(err, "failed to write properties %s to backend", keyProperties.String())
	}
	return nil

}

type backendHandler struct {
	Properties map[string][]string
	Editor     editor.Interface
}
