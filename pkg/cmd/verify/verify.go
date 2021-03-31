package verify

import (
	"fmt"
	"os"
	"strings"

	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/table"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x-plugins/jx-secret/pkg/extsecrets/secretfacade"
	"github.com/jenkins-x-plugins/jx-secret/pkg/rootcmd"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	verifyLong = templates.LongDesc(`
		Verifies that the ExternalSecret resources have the required properties populated in the underlying secret storage
`)

	verifyExample = templates.Examples(`
		%s verify
	`)
)

// Options the options for the command
type Options struct {
	secretfacade.Options

	Results []*secretfacade.SecretError
}

// NewCmdVerify creates a command object for the command
func NewCmdVerify() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "verify",
		Aliases: []string{"get"},
		Short:   "Verifies that the ExternalSecret resources have the required properties populated in the underlying secret storage",
		Long:    verifyLong,
		Example: fmt.Sprintf(verifyExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "", "the namespace to filter the ExternalSecret resources")
	o.Options.AddFlags(cmd)
	return cmd, o
}

// Run implements the command
func (o *Options) Run() error {
	err := o.Validate()
	if err != nil {
		return errors.Wrap(err, "error validating options")
	}

	pairs, err := o.Verify()
	if err != nil {
		return errors.Wrap(err, "failed to verify secrets")
	}
	o.Results = nil

	t := table.CreateTable(os.Stdout)
	t.AddRow("SECRET", "STATUS")
	for _, r := range pairs {
		name := r.ExternalSecret.Name
		state := r.Error
		ns := r.ExternalSecret.Namespace
		fullName := name
		if ns != "" && o.Namespace == "" {
			fullName = ns + "/" + name
		}
		if state == nil {
			t.AddRow(fullName, termcolor.ColorInfo(fmt.Sprintf("valid: %s", strings.Join(r.ExternalSecret.KeyAndNames(), ", "))))
		} else {
			o.Results = append(o.Results, state)
			for _, e := range state.EntryErrors {
				t.AddRow(fullName, termcolor.ColorWarning(fmt.Sprintf("key %s missing properties: %s", e.Key, strings.Join(e.Properties, ", "))))
			}
		}
	}
	t.Render()
	return nil
}
