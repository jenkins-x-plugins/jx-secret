package portforward

import (
	"fmt"

	"github.com/jenkins-x-plugins/jx-secret/pkg/cmd/vault/wait"
	"github.com/jenkins-x-plugins/jx-secret/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	cmdLong = templates.LongDesc(`
		Runs a port forward process so you can access the vault in a kubernetes cluster
`)

	cmdExample = templates.Examples(`
		%s vault portforward
	`)
)

// Options the options for the command
type Options struct {
	wait.Options
	CommandRunner cmdrunner.CommandRunner
}

// NewCmdPortForward creates a command object for the command
func NewCmdPortForward() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "portforward",
		Short:   "Runs a port forward process so you can access the vault in a kubernetes cluster",
		Aliases: []string{"portfwd", "port-forward"},
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	o.Options.AddFlags(cmd)
	return cmd, o
}

// Run implements the command
func (o *Options) Run() error {
	o.Options.NoEditorWait = true
	err := o.Options.Run()
	if err != nil {
		return errors.Wrapf(err, "failed to wait for vault")
	}
	if o.CommandRunner == nil {
		o.CommandRunner = cmdrunner.DefaultCommandRunner
	}
	cmd := &cmdrunner.Command{
		Name: "kubectl",
		Args: []string{"port-forward", "--namespace", o.Namespace, "service/vault", "8200"},
	}
	_, err = o.CommandRunner(cmd)
	if err != nil {
		return errors.Wrapf(err, "failed to run command: %s", cmd.CLI())
	}
	return nil
}
