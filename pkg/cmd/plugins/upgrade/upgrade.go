package upgrade

import (
	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/templates"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/jenkins-x/jx-secret/pkg/plugins"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	cmdLong = templates.LongDesc(`
		Upgrades the binary plugins of the secret command (e.g. the vault binary)
`)

	cmdExample = templates.Examples(`
		# upgrades the plugin binaries
		jx upgrade
	`)
)

// UpgradeOptions the options for upgrading a cluster
type UpgradeOptions struct {
	CommandRunner cmdrunner.CommandRunner
}

// NewCmdUpgrade creates a command object for the command
func NewCmdUpgrade() (*cobra.Command, *UpgradeOptions) {
	o := &UpgradeOptions{}

	cmd := &cobra.Command{
		Use:     "upgrade",
		Short:   "Upgrades the binary plugins of the secret command (e.g. the Vault binary)",
		Long:    cmdLong,
		Example: cmdExample,
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	return cmd, o
}

// Run implements the command
func (o *UpgradeOptions) Run() error {
	log.Logger().Infof("checking we have the correct vault CLI version")
	_, err := plugins.GetVaultBinary("")
	if err != nil {
		return errors.Wrapf(err, "failed to check vault binary")
	}
	return nil
}
