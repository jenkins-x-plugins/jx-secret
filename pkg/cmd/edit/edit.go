package edit

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/jenkins-x-plugins/jx-secret/pkg/apis/mapping/v1alpha1"
	"github.com/jenkins-x-plugins/jx-secret/pkg/cmd/populate"
	"github.com/jenkins-x/jx-kube-client/v3/pkg/kubeclient"

	v1 "github.com/jenkins-x-plugins/jx-secret/pkg/apis/external/v1"
	schemaapi "github.com/jenkins-x-plugins/jx-secret/pkg/apis/schema/v1alpha1"
	"github.com/jenkins-x-plugins/jx-secret/pkg/extsecrets/editor"
	"github.com/jenkins-x-plugins/jx-secret/pkg/extsecrets/editor/factory"
	"github.com/jenkins-x-plugins/jx-secret/pkg/extsecrets/secretfacade"
	"github.com/jenkins-x-plugins/jx-secret/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/input"
	"github.com/jenkins-x/jx-helpers/v3/pkg/input/survey"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	cmdLong = templates.LongDesc(`
		Edits secret values in the underlying secret stores for ExternalSecrets
`)

	cmdExample = templates.Examples(`
		# edit any missing mandatory secrets
		%s edit

		# edit any secrets with a given filter
		%s edit --filter nexus
	`)
)

// Options the options for the command
type Options struct {
	secretfacade.Options
	Filter               string
	Interactive          bool
	InteractiveMultiple  bool
	InteractiveSelectAll bool
	ExternalVault        string
	Input                input.Interface
	Results              []*secretfacade.SecretPair
	CommandRunner        cmdrunner.CommandRunner
	QuietCommandRunner   cmdrunner.CommandRunner
}

// NewCmdEdit creates a command object for the command
func NewCmdEdit() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "edit",
		Short:   "Edits secret values in the underlying secret stores for ExternalSecrets",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Namespace, "ns", "n", "", "the namespace to filter the ExternalSecret resources")
	cmd.Flags().StringVarP(&o.Dir, "dir", "d", ".", "the directory to look for the .jx/secret/mapping/secret-mappings.yaml file")
	cmd.Flags().StringVarP(&o.Filter, "filter", "f", "", "filter on the Secret / ExternalSecret names to enter")
	cmd.Flags().BoolVarP(&o.Interactive, "interactive", "i", false, "interactive mode asks the user for the Secret name and the properties to edit")
	cmd.Flags().BoolVarP(&o.InteractiveMultiple, "multiple", "m", false, "for interactive mode do you want to select multiple secrets to edit. If not defaults to just picking a single secret")
	cmd.Flags().BoolVarP(&o.InteractiveSelectAll, "all", "", false, "for interactive mode do you want to select all of the properties to edit by default. Otherwise none are selected and you choose to select the properties to change")
	cmd.Flags().StringVarP(&o.ExternalVault, "external-vault", "", os.Getenv("EXTERNAL_VAULT"), "specify whether we are using external vault or not")
	return cmd, o
}

// Run implements the command
func (o *Options) Run() error {
	err := o.Validate()
	if err != nil {
		return errors.Wrap(err, "error validating options")
	}
	// it only makes sense to choose secrets from all namespaces when not having -f or -i., ie when you want to set all unset properties
	if o.Namespace == "" && (o.Filter != "" || o.Interactive) {
		o.Namespace, err = kubeclient.CurrentNamespace()
		if err != nil {
			log.Logger().Warnf("failed to get current namespace, defaulting to all: %s", err.Error())
		}
	}
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

	if o.Input == nil {
		o.Input = survey.NewInput()
	}

	if o.Interactive {
		results, err = o.chooseSecrets(results)
		if err != nil {
			return errors.Wrapf(err, "failed to choose secrets in interactive mode")
		}
	}

	for i := range results {
		r := results[i]
		name := r.ExternalSecret.Name
		secEditor, err := factory.NewEditor(&r.ExternalSecret, o.SecretStoreManagerFactory, o.KubeClient, o.ExternalVault)
		if err != nil {
			return errors.Wrapf(err, "failed to create a secret editor for ExternalSecret %s", name)
		}

		// todo do we need to find any surveys that require a confirm?
		// order them somehow?
		// maybe skip any?
		if o.Matches(r) {
			data := o.DataToEdit(r)

			m := map[string]*editor.KeyProperties{}
			for i := range data {
				d := &data[i]
				key := populate.GetSecretKey(v1alpha1.BackendType(r.ExternalSecret.Spec.BackendType), name, d.Key)
				property := d.Property

				var value string
				value, err = o.askForSecretValue(r, d)
				if err != nil {
					if errors.Is(err, terminal.InterruptErr) {
						// interpreting interrupt (i.e. ctrl+c) as skipping property
						log.Logger().Infof("skipping property %s", property)
						continue
					}
					return errors.Wrapf(err, "failed to ask user secret value property %s for key %s on ExternalSecret %s", property, key, name)
				}
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

				// fix issue where strings with newlines were being escaped when being marshalled later, so let's ensure newlines are used
				// see https://stackoverflow.com/questions/32042989/go-lang-differentiate-n-and-line-break
				value = strings.Replace(value, `\n`, "\n", -1)

				keyProperties.Properties = append(keyProperties.Properties, editor.PropertyValue{
					Property: property,
					Value:    value,
				})

			}
			for _, keyProperties := range m {
				err = secEditor.Write(keyProperties)
				if err != nil {
					return errors.Wrapf(err, "failed to save properties %s on ExternalSecret %s", keyProperties.String(), name)
				}
			}
		}
	}
	return nil
}

func (o *Options) chooseSecrets(results []*secretfacade.SecretPair) ([]*secretfacade.SecretPair, error) {
	var names []string
	m := map[string][]*secretfacade.SecretPair{}
	for _, s := range results {
		name := s.ExternalSecret.Name
		if len(m[name]) == 0 {
			names = append(names, name)
		}
		m[name] = append(m[name], s)
	}
	sort.Strings(names)

	var err error
	if o.InteractiveMultiple {
		names, err = o.Input.SelectNames(names, "Pick the Secrets to edit", false, "select the names of the ExternalSecrets you want to edit")
		if err != nil {
			return nil, errors.Wrapf(err, "failed to ")
		}
		if len(names) == 0 {
			return nil, errors.Errorf("no ExternalSecret names selected")
		}
	} else {
		name := ""
		name, err = o.Input.PickNameWithDefault(names, "Pick the Secret to edit", "", "select the name of the ExternalSecrets you want to edit")
		if err != nil {
			return nil, errors.Wrapf(err, "failed to ")
		}
		if name == "" {
			return nil, errors.Errorf("no ExternalSecret name selected")
		}
		names = []string{name}
	}
	var answer []*secretfacade.SecretPair
	for _, name := range names {
		answer = append(answer, m[name]...)
	}
	return answer, nil
}

func (o *Options) askForSecretValue(s *secretfacade.SecretPair, d *v1.Data) (string, error) {
	var value string
	var err error
	name := s.ExternalSecret.Name
	property := d.Property
	object, err := s.SchemaObject()
	if err != nil {
		return "", errors.Wrapf(err, "failed to find object schema for object %s property %s", name, property)
	}
	propertySpec := object.FindProperty(d.Name)
	if propertySpec == nil {
		message, help := o.propertyMessage(s, d)
		value, err = o.Input.PickPassword(message, help) //nolint:govet
		if err != nil {
			return "", errors.Wrapf(err, "failed to enter property %s for key %s on ExternalSecret %s", property, d.Key, name)
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

	kind := propertySpec.Labels[schemaapi.LabelKind]
	switch kind {
	case "confirm":
		log.Logger().Warn("implement confirm")
	default:
		return o.Input.PickPassword(propertySpec.Question, propertySpec.Help) //nolint:govet
	}
	return value, nil
}

func (o *Options) propertyMessage(s *secretfacade.SecretPair, d *v1.Data) (string, string) {
	name := s.ExternalSecret.Name
	property := d.Property
	if property == "" {
		property = d.Name
	}
	return name + "." + property, ""
}

// Matches returns true if the secret matches the current filter
// If no filter then just filter out mandatory properties only?
func (o *Options) Matches(r *secretfacade.SecretPair) bool {
	if o.Filter == "" {
		if o.Interactive {
			return true
		}
		return r.IsInvalid()
	}
	return strings.Contains(r.ExternalSecret.Name, o.Filter)
}

// DataToEdit returns the properties to edit
func (o *Options) DataToEdit(r *secretfacade.SecretPair) []v1.Data {
	if o.Interactive {
		var names []string
		m := map[string]*v1.Data{}
		for i := range r.ExternalSecret.Spec.Data {
			data := &r.ExternalSecret.Spec.Data[i]
			name := data.Name
			names = append(names, name)
			m[name] = data
		}

		var err error
		names, err = o.Input.SelectNames(names, "Pick the secret properties to edit: ", o.InteractiveSelectAll, "Please choose the names to edit in the ExternalSecret")
		if err != nil {
			log.Logger().Warnf("failed to pick the data entries to edit: %s", err.Error())
		}

		var answer []v1.Data
		for _, name := range names {
			answer = append(answer, *m[name])
		}
		return answer
	}

	// if filtering return all properties
	if o.Filter != "" {
		return r.ExternalSecret.Spec.Data
	}

	missingProperties := map[string]bool{}
	if r.Error != nil {
		for _, e := range r.Error.EntryErrors {
			for _, n := range e.Properties {
				missingProperties[n] = true
			}
		}
	}

	// otherwise return only missing fields
	var results []v1.Data
	for _, d := range r.ExternalSecret.Spec.Data {
		if missingProperties[d.Property] {
			results = append(results, d)
		}
	}
	return results
}

func (o *Options) VerifyAndFilter() ([]*secretfacade.SecretPair, error) {
	secrets, err := o.Verify()
	if err != nil {
		return secrets, err
	}

	// let's filter out any secrets with same locations...
	destinations := map[string][]*secretfacade.SecretPair{}

	for _, s := range secrets {
		es := s.ExternalSecret
		for _, property := range es.Spec.Data {
			destination := property.SecretLocation(es.Spec.BackendType)
			destinations[destination] = append(destinations[destination], s)
		}
	}

	filterKeys := map[string]bool{}
	for destination, secretsForDestination := range destinations {
		if len(secretsForDestination) < 2 {
			continue
		}

		secretfacade.SortSecretsInSchemaOrder(secretsForDestination)
		for i := 1; i < len(secretsForDestination); i++ {
			key := secretsForDestination[i].Key()
			if !filterKeys[key] {
				log.Logger().Debugf("filtering out Secret %s/%s as %s/%s is better for schema editing and it uses the same destination %s", secretsForDestination[i].Namespace(), secretsForDestination[i].Name(), secretsForDestination[0].Namespace(), secretsForDestination[0].Name(), destination)
				filterKeys[key] = true
			}
		}
	}

	var answer []*secretfacade.SecretPair
	for _, s := range secrets {
		key := s.Key()
		if !filterKeys[key] {
			answer = append(answer, s)
		}
	}
	return answer, nil
}
