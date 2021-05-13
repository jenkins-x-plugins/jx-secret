package postrender

import (
	"fmt"
	"github.com/jenkins-x-plugins/jx-secret/pkg/cmd/convert"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kyamls"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"sigs.k8s.io/kustomize/kyaml/yaml"
	"strings"

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
	ConvertOptions convert.Options
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

	// lets transform....
	_, err = o.ConvertOptions.ModifyYAML(node, path)
	if err != nil {
		return text, errors.Wrapf(err, "failed to convert Secret")
	}

	// now lets save the modified node
	out, err := node.String()
	if err != nil {
		return "", errors.Wrapf(err, "failed to marshal converted YAML")
	}
	return out, nil
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
