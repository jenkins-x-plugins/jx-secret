package upgrade

import (
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
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
type Options struct {
	CommandRunner cmdrunner.CommandRunner
}

// NewCmdUpgrade creates a command object for the command
func NewCmdUpgrade() (*cobra.Command, *Options) {
	o := &Options{}

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
func (o *Options) Run() error {
	log.Logger().Infof("checking we have the correct vault CLI version")
	_, err := plugins.GetVaultBinary("")
	if err != nil {
		return errors.Wrapf(err, "failed to check vault binary")
	}
	return nil
}
