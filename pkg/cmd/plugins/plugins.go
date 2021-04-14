package plugins

import (
	"github.com/jenkins-x-plugins/jx-secret/pkg/cmd/plugins/upgrade"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/spf13/cobra"
)

// NewCmdPlugins creates the new command
func NewCmdPlugins() *cobra.Command {
	command := &cobra.Command{
		Use:     "plugins",
		Aliases: []string{"plugin"},
		Short:   "Commands for working with Plugins",
		Run: func(command *cobra.Command, args []string) {
			err := command.Help()
			if err != nil {
				log.Logger().Errorf(err.Error())
			}
		},
	}
	command.AddCommand(cobras.SplitCommand(upgrade.NewCmdUpgrade()))
	return command
}
