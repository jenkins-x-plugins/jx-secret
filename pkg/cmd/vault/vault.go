package vault

import (
	"github.com/jenkins-x/jx-helpers/pkg/cobras"
	"github.com/jenkins-x/jx-logging/pkg/log"
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
	command.AddCommand(cobras.SplitCommand(NewCmdPortForward()))
	return command
}
