package populate

import (
	"fmt"
	"path/filepath"

	"github.com/jenkins-x/jx-helpers/pkg/stringhelpers"
	"github.com/jenkins-x/jx-secret/pkg/schema"
	"github.com/jenkins-x/jx-secret/pkg/schema/secrets"

	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets/editor"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets/editor/factory"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets/secretfacade"
	"github.com/jenkins-x/jx-secret/pkg/root"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	cmdLong = templates.LongDesc(`
		Populates any missing secret values which can be automatically generated"
`)

	cmdExample = templates.Examples(`
		%s populate
	`)
)

// Options the options for the command
type Options struct {
	secretfacade.Options

	Dir           string
	Schema        *schema.Schema
	Results       []*secretfacade.SecretError
	CommandRunner cmdrunner.CommandRunner
}

// NewCmdPopulate creates a command object for the command
func NewCmdPopulate() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "populate",
		Short:   "Populates any missing secret values which can be automatically generated",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, root.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Namespace, "ns", "n", "", "the namespace to filter the ExternalSecret resources")
	cmd.Flags().StringVarP(&o.Dir, "dir", "d", ".", "the directory to look for the .jx/gitops/secret-schema.yaml file")
	return cmd, o
}

// Run implements the command
func (o *Options) Run() error {
	// get a list of external secrets which do not have corresponding k8s secret data populated
	results, err := o.Verify()
	if err != nil {
		return errors.Wrap(err, "failed to verify secrets")
	}
	o.Results = results

	if len(results) == 0 {
		log.Logger().Infof("the %d ExternalSecrets are %s", len(o.ExternalSecrets), termcolor.ColorInfo("populated"))
		return nil
	}

	editors := map[string]editor.Interface{}

	o.Schema, err = schema.LoadSchema(filepath.Join(o.Dir, ".jx", "gitops", "secret-schema.yaml"))
	if err != nil {
		return errors.Wrapf(err, "failed to load survey schema used to prompt the user for questions")
	}
	for _, r := range results {
		name := r.ExternalSecret.Name
		backendType := r.ExternalSecret.Spec.BackendType
		secEditor := editors[backendType]
		log.Logger().Infof("using %s as the secrets store", backendType)
		if secEditor == nil {
			secEditor, err = factory.NewEditor(&r.ExternalSecret, o.CommandRunner, o.KubeClient)
			if err != nil {
				return errors.Wrapf(err, "failed to create a secret editor for ExternalSecret %s", name)
			}
			editors[backendType] = secEditor
		}

		for _, e := range r.EntryErrors {
			keyProperties := editor.KeyProperties{
				Key: e.Key,
			}
			for _, property := range e.Properties {
				var value string
				value, err = o.generateSecretValue(name, property, e)
				if err != nil {
					return errors.Wrapf(err, "failed to ask user secret value property %s for key %s on ExternalSecret %s", property, e.Key, name)
				}
				if value == "" {
					continue
				}
				keyProperties.Properties = append(keyProperties.Properties, editor.PropertyValue{
					Property: property,
					Value:    value,
				})
			}

			if len(keyProperties.Properties) > 0 {
				err = secEditor.Write(keyProperties)
				if err != nil {
					return errors.Wrapf(err, "failed to save properties %s on ExternalSecret %s", keyProperties.String(), name)
				}
			}
		}
	}
	return nil
}

func (o *Options) generateSecretValue(secretName, property string, e *secretfacade.EntryError) (string, error) {
	propertySchema, err := schema.FindObjectProperty(o.Schema, secretName, property)
	if err != nil {
		return "", errors.Wrapf(err, "failed to find schema for entry %s property %s", e.Key, property)
	}
	if propertySchema == nil {
		return "", nil
	}

	if propertySchema.DefaultValue != "" {
		return propertySchema.DefaultValue, nil
	}

	if propertySchema.Format == "hmac" {
		value, err := stringhelpers.RandStringBytesMaskImprSrc(41)
		if err != nil {
			return value, errors.Wrapf(err, "generating hmac")
		}
		return value, nil
	}

	// if can generate then use generator
	if propertySchema.Generate {
		length := propertySchema.MaxLength
		if length == 0 {
			length = propertySchema.MinLength
			if length == 0 {
				length = 20
			}
		}
		value, err := secrets.DefaultGenerateSecret(length)
		if err != nil {
			return value, errors.WithStack(err)
		}
		return value, nil
	}
	return "", nil
}
