package edit

import (
	"fmt"

	"github.com/jenkins-x/jx-extsecret/pkg/cmdrunner"
	"github.com/jenkins-x/jx-extsecret/pkg/extsecrets/editor"
	"github.com/jenkins-x/jx-extsecret/pkg/extsecrets/editor/factory"
	"github.com/jenkins-x/jx-extsecret/pkg/extsecrets/secretfacade"
	"github.com/jenkins-x/jx-extsecret/pkg/input"
	"github.com/jenkins-x/jx-extsecret/pkg/input/survey"
	"github.com/jenkins-x/jx-extsecret/pkg/root"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/jenkins-x/jx/v2/pkg/cmd/helper"
	"github.com/jenkins-x/jx/v2/pkg/cmd/templates"
	"github.com/jenkins-x/jx/v2/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	editLong = templates.LongDesc(`
		Edits any missing properties in the ExternalSecret resources
`)

	editExample = templates.Examples(`
		%s edit
	`)
)

// Options the options for the command
type Options struct {
	secretfacade.Options

	Input         input.Interface
	Results       []*secretfacade.SecretError
	CommandRunner cmdrunner.CommandRunner
}

// NewCmdEdit creates a command object for the command
func NewCmdEdit() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "edit",
		Short:   "Edits any missing properties in the ExternalSecret resources",
		Long:    editLong,
		Example: fmt.Sprintf(editExample, root.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Namespace, "ns", "n", "", "the namespace to filter the ExternalSecret resources")
	return cmd, o
}

// Run implements the command
func (o *Options) Run() error {
	results, err := o.Verify()
	if err != nil {
		return errors.Wrap(err, "failed to verify secrets")
	}
	o.Results = results

	if len(results) == 0 {
		log.Logger().Infof("the %d ExternalSecrets are %s", len(o.ExternalSecrets), util.ColorInfo("valid"))
		return nil
	}

	if o.Input == nil {
		o.Input = survey.NewInput()
	}

	for _, r := range results {
		name := r.ExternalSecret.Name
		secEditor, err := factory.NewEditor(&r.ExternalSecret, o.CommandRunner)
		if err != nil {
			return errors.Wrapf(err, "failed to create a secret editor for ExternalSecret %s", name)
		}

		for _, e := range r.EntryErrors {
			keyProperties := editor.KeyProperties{
				Key: e.Key,
			}
			for _, property := range e.Properties {
				message, help := o.propertyMessage(r, e, property)
				value, err := o.Input.PickPassword(message, help)
				if err != nil {
					return errors.Wrapf(err, "failed to enter property %s for key %s on ExternalSecret %s", property, e.Key, name)
				}
				keyProperties.Properties = append(keyProperties.Properties, editor.PropertyValue{
					Property: property,
					Value:    value,
				})
			}

			err = secEditor.Write(keyProperties)
			if err != nil {
				return errors.Wrapf(err, "failed to save properties %s on ExternalSecret %s", keyProperties.String(), name)
			}

		}
	}
	return nil
}

func (o *Options) propertyMessage(r *secretfacade.SecretError, e *secretfacade.EntryError, property string) (string, string) {
	return e.Key + "." + property, ""
}
