package vault

import (
	"github.com/jenkins-x/jx-helpers/pkg/cobras"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/jenkins-x/jx-secret/pkg/cmd/vault/portforward"
	"github.com/jenkins-x/jx-secret/pkg/cmd/vault/wait"
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
	command.AddCommand(cobras.SplitCommand(wait.NewCmdWait()))
	command.AddCommand(cobras.SplitCommand(portforward.NewCmdPortForward()))
	return command
}
