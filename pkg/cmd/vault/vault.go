package vault

import (
	"github.com/jenkins-x-plugins/jx-secret/pkg/cmd/vault/portforward"
	"github.com/jenkins-x-plugins/jx-secret/pkg/cmd/vault/shell"
	"github.com/jenkins-x-plugins/jx-secret/pkg/cmd/vault/wait"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/spf13/cobra"
)

// NewCmdVault creates the new command
func NewCmdVault() *cobra.Command {
	command := &cobra.Command{
		Use:   "vault",
		Short: "Commands for working with Vault",
		Run: func(command *cobra.Command, args []string) {
			err := command.Help()
			if err != nil {
				log.Logger().Errorf(err.Error())
			}
		},
	}
	command.AddCommand(cobras.SplitCommand(portforward.NewCmdPortForward()))
	command.AddCommand(cobras.SplitCommand(shell.NewCmdVaultShell()))
	command.AddCommand(cobras.SplitCommand(wait.NewCmdWait()))
	return command
}
