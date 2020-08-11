package edit

import (
	"fmt"
	"path/filepath"

	"github.com/jenkins-x/jx-secret/pkg/schema"

	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/pkg/input"
	"github.com/jenkins-x/jx-helpers/pkg/input/survey"
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
	Survey        *schema.SurveySchema
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

	if o.Input == nil {
		o.Input = survey.NewInput()
	}

	editors := map[string]editor.Interface{}

	o.Survey, err = schema.LoadSurveySchema(filepath.Join(".jx", "gitops", "secret-schema.yaml"))
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

		// todo do we need to find any surveys that require a confirm?
		// order them somehow?
		// maybe skip any?
		for _, e := range r.EntryErrors {
			keyProperties := editor.KeyProperties{
				Key: e.Key,
			}
			for _, property := range e.Properties {
				var value string
				value, err = o.askForSecretValue(e, property, name)
				if err != nil {
					return errors.Wrapf(err, "failed to ask user secret value property %s for key %s on ExternalSecret %s", property, e.Key, name)
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

func (o *Options) propertyMessage(e *secretfacade.EntryError, property string) (string, string) {
	return e.Key + "." + property, ""
}

func (o *Options) askForSecretValue(e *secretfacade.EntryError, property, name string) (string, error) {
	var value string
	var err error
	var surveySchema schema.Survey

	surveySchema, err = o.findSurveyForSecret(e, property)
	if err != nil {
		message, help := o.propertyMessage(e, property)
		value, err = o.Input.PickPassword(message, help) //nolint:govet
		if err != nil {
			return "", errors.Wrapf(err, "failed to enter property %s for key %s on ExternalSecret %s", property, e.Key, name)
		}
		return value, nil
	}

	// if mask

	// if format

	// if pattern?

	// min / max

	// if confirm

	// if git get the kind URL / template the help and question?

	// Add TESTS!!!

	kind := surveySchema.Labels[schema.LabelKind]
	switch kind {
	case "confirm":
		log.Logger().Warn("implement confirm")
	default:
		return o.Input.PickPassword(surveySchema.Question, surveySchema.Help) //nolint:govet
	}
	return value, nil
}

func (o *Options) findSurveyForSecret(e *secretfacade.EntryError, property string) (schema.Survey, error) {
	answer := schema.Survey{}
	if o.Survey != nil {
		return answer, errors.New("no surveys found")
	}
	for i := range o.Survey.Spec.Survey {
		s := &o.Survey.Spec.Survey[i]

		// match using labels, first on the secretKey and next on the secretProperty if one exists
		if s.Labels[schema.LabelSecretKey] == e.Key {
			if property == "" {
				return *s, nil
			} else if s.Labels[schema.LabelSecretProperty] == property {
				return *s, nil
			}
		}
	}
	return answer, errors.New("no matching surveys found")
}
