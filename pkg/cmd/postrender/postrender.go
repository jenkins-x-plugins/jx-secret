package postrender

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/jenkins-x-plugins/jx-secret/pkg/cmd/convert"
	"github.com/jenkins-x-plugins/jx-secret/pkg/cmd/populate"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kyamls"
	"github.com/jenkins-x/jx-helpers/v3/pkg/options"
	"github.com/pkg/errors"
	"github.com/sethvargo/go-envconfig"
	"sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/jenkins-x-plugins/jx-secret/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/spf13/cobra"
)

var (
	// resourcesSeparator is used to separate multiple objects stored in the same YAML file
	resourcesSeparator = "---\n"

	cmdLong = templates.LongDesc(`
		A helm postrender to convert any Secret resources into ExternalSecret resources
`)

	cmdExample = templates.Examples(`
		# lets post render
		helm install --postrender 'jx secret postrender'  myname mychart
	`)

	secretFilter = kyamls.ParseKindFilter("v1/Secret")
)

// Options the options for the command
type Options struct {
	EnvOptions

	ConvertOptions  convert.Options
	PopulateOptions populate.Options

	SecretCount int
}

type EnvOptions struct {
	options.BaseOptions

	VaultMountPoint  string `env:"JX_VAULT_MOUNT_POINT"`
	VaultRole        string `env:"JX_VAULT_ROLE"`
	Dir              string `env:"JX_DIR"`
	DefaultNamespace string `env:"JX_DEFAULT_NAMESPACE"`

	// DisablePopulate disables if external secrets are populated from the helm secret data if not populated already
	DisablePopulate bool `env:"JX_NO_POPULATE"`
}

// NewCmdPostrender creates a command object for the command
func NewCmdPostrender() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "postrender",
		Short:   "A helm postrender to convert any Secret resources into ExternalSecret resources",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	return cmd, o
}

// Run implements the command
func (o *Options) Run() error {
	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return errors.Wrapf(err, "failed to read standard input")
	}

	ctx := context.TODO()
	err = envconfig.Process(ctx, &o.EnvOptions)
	if err != nil {
		return errors.Wrapf(err, "failed to process environment options")
	}

	o.ConvertOptions.BaseOptions = o.EnvOptions.BaseOptions
	o.ConvertOptions.BatchMode = true
	o.ConvertOptions.DefaultNamespace = o.EnvOptions.DefaultNamespace
	o.ConvertOptions.Dir = o.EnvOptions.Dir
	o.ConvertOptions.VaultMountPoint = o.EnvOptions.VaultMountPoint
	o.ConvertOptions.VaultRole = o.EnvOptions.VaultRole
	o.PopulateOptions.Options.BaseOptions = o.EnvOptions.BaseOptions
	o.PopulateOptions.Options.BatchMode = true
	o.PopulateOptions.DisableSecretFolder = true

	err = o.ConvertOptions.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to validate options")
	}

	sections := strings.Split(string(data), resourcesSeparator)

	buf := &strings.Builder{}

	for i, section := range sections {
		if i > 0 {
			buf.WriteString("\n")
			buf.WriteString(resourcesSeparator)
		}
		if IsWhitespaceOrComments(section) {
			buf.WriteString(section)
			continue
		}
		result, err := o.Convert(section)
		if err != nil {
			return errors.Wrapf(err, "failed to convert section")
		}
		buf.WriteString(result)
	}

	if o.SecretCount > 0 && !o.DisablePopulate {
		err = o.PopulateSecrets()
		if err != nil {
			return errors.Wrapf(err, "failed to ")
		}
		buf.WriteString(fmt.Sprintf("\n\n# failed to populate external secret store\n# %s\n", err.Error()))
	}
	fmt.Println(buf.String())
	return nil
}

func (o *Options) Convert(text string) (string, error) {
	path := ""
	node, err := yaml.Parse(text)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse YAML")
	}
	if !secretFilter.Matches(node, path) {
		return text, nil
	}

	secretData, err := o.GetSecretData(node, path)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get secret data")
	}

	// lets transform....
	opts, err := o.ConvertOptions.ModifyYAML(node, path)
	if err != nil {
		return text, errors.Wrapf(err, "failed to convert Secret")
	}

	// populate the secret data so we can lazy populate any external secret store
	if secretData != nil {
		key := scm.Join(opts.Namespace, opts.Name)
		if o.PopulateOptions.HelmSecretValues == nil {
			o.PopulateOptions.HelmSecretValues = map[string]map[string]string{}
		}
		o.PopulateOptions.HelmSecretValues[key] = secretData
	}

	// lets save the modified node
	out, err := node.String()
	if err != nil {
		return "", errors.Wrapf(err, "failed to marshal converted YAML")
	}
	o.SecretCount++
	return out, nil
}

func (o *Options) PopulateSecrets() error {
	err := o.PopulateOptions.Run()
	if err != nil {
		return errors.Wrapf(err, "failed to populate serets")
	}
	return nil
}

func (o *Options) GetSecretData(node *yaml.RNode, path string) (map[string]string, error) {
	m := map[string]string{}
	for _, dataPath := range []string{"data", "stringData"} {
		data, err := node.Pipe(yaml.Lookup(dataPath))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get data for path %s", path)
		}

		if data != nil {
			fields, err := data.Fields()
			if err != nil {
				return nil, errors.Wrapf(err, "failed to find data fields for path %s", path)
			}

			for _, field := range fields {
				//if o.SecretMapping.IsSecretKeyUnsecured(secretName, field) {
				value := kyamls.GetStringField(data, "", field)
				if value == "" {
					continue
				}
				m[field] = value
			}
		}
	}
	return m, nil
}

// IsWhitespaceOrComments returns true if the text is empty, whitespace or comments only
func IsWhitespaceOrComments(text string) bool {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if t != "" && !strings.HasPrefix(t, "#") && !strings.HasPrefix(t, "--") {
			return false
		}
	}
	return true
}
