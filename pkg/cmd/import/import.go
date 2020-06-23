package importcmd

import (
	"fmt"
	"io/ioutil"

	"github.com/go-yaml/yaml"
	"github.com/jenkins-x/jx-extsecret/pkg/apis/extsecret/v1alpha1"
	"github.com/jenkins-x/jx-extsecret/pkg/cmdrunner"
	"github.com/jenkins-x/jx-extsecret/pkg/extsecrets"
	"github.com/jenkins-x/jx-extsecret/pkg/extsecrets/editor"
	"github.com/jenkins-x/jx-extsecret/pkg/extsecrets/editor/factory"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/jenkins-x/jx-promote/pkg/common"
	"github.com/jenkins-x/jx/v2/pkg/cmd/helper"
	"github.com/jenkins-x/jx/v2/pkg/cmd/templates"
	"github.com/jenkins-x/jx/v2/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	editLong = templates.LongDesc(`
		Imports a YAML of secret values into the underlying secret store
`)

	editExample = templates.Examples(`
		%s edit
	`)
)

// Options the options for the command
type Options struct {
	File             string
	Namespace        string
	SecretClient     extsecrets.Interface
	CommandRunner    cmdrunner.CommandRunner
	FailOnUnknownKey bool
	ExternalSecrets  []*v1alpha1.ExternalSecret
	Handlers         map[string]*backendHandler
}

// NewCmdImport creates a command object for the command
func NewCmdImport() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "import",
		Short:   "Imports a YAML file of secret values",
		Long:    editLong,
		Example: fmt.Sprintf(editExample, common.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Namespace, "ns", "n", "", "the namespace to filter the ExternalSecret resources")
	cmd.Flags().BoolVarP(&o.FailOnUnknownKey, "fail-on-unknown-key", "", false, "should the command fail if a key from the YAML file is unknown")
	return cmd, o
}

// Run implements the command
func (o *Options) Run() error {
	fileName := o.File
	if fileName == "" {
		return util.MissingOption("file")
	}

	m := map[interface{}]interface{}{}
	exists, err := util.FileExists(fileName)
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

	resources, err := o.SecretClient.List(o.Namespace, metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to find external secrets")
	}
	o.ExternalSecrets = resources

	log.Logger().Debugf("found %d ExternalSecret resources", len(resources))

	for _, r := range resources {
		ns := r.Namespace
		if ns == "" {
			ns = o.Namespace
		}
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
				editor, err := factory.NewEditor(r, o.CommandRunner)
				if err != nil {
					return errors.Wrapf(err, "failed to create editor for secret %s of type %s", name, backendType)
				}
				handler = &backendHandler{
					Properties: map[string][]string{},
					Editor:     editor,
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

func (o *Options) updateProperties(m map[interface{}]interface{}, path string) error {
	keyProperties := editor.KeyProperties{
		Key: path,
	}
	for k, v := range m {
		key := fmt.Sprintf("%v", k)
		childPath := key
		if path != "" {
			childPath = path + "/" + key
		}
		childMap, ok := v.(map[interface{}]interface{})
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
			Property: key,
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
