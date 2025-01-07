package cmd

import (
	"github.com/jenkins-x-plugins/jx-secret/pkg/cmd/convert"
	"github.com/jenkins-x-plugins/jx-secret/pkg/cmd/copy"
	"github.com/jenkins-x-plugins/jx-secret/pkg/cmd/edit"
	"github.com/jenkins-x-plugins/jx-secret/pkg/cmd/plugins"
	"github.com/jenkins-x-plugins/jx-secret/pkg/cmd/populate"
	"github.com/jenkins-x-plugins/jx-secret/pkg/cmd/replicate"
	"github.com/jenkins-x-plugins/jx-secret/pkg/cmd/vault"
	"github.com/jenkins-x-plugins/jx-secret/pkg/cmd/verify"
	"github.com/jenkins-x-plugins/jx-secret/pkg/cmd/version"
	"github.com/jenkins-x-plugins/jx-secret/pkg/cmd/wait"
	"github.com/jenkins-x-plugins/jx-secret/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/spf13/cobra"
)

var secretRetriableErrors = []string{
	"dial tcp \\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}:\\d+: i/o timeout",
}

// Main creates the new command
func Main() *cobra.Command {
	cmd := &cobra.Command{
		Use:   rootcmd.TopLevelCommand,
		Short: "commands for working with Secrets, ExternalSecrets and external secret stores",
		Run: func(cmd *cobra.Command, args []string) {
			err := cmd.Help()
			if err != nil {
				log.Logger().Error(err.Error())
			}
		},
	}
	cmd.AddCommand(cobras.SplitCommand(convert.NewCmdSecretConvert()))
	cmd.AddCommand(cobras.SplitCommand(copy.NewCmdCopy()))
	cmd.AddCommand(cobras.SplitCommand(edit.NewCmdEdit()))
	cmd.AddCommand(helper.RetryOnErrorCommand(cobras.SplitCommand(populate.NewCmdPopulate()), helper.RegexRetryFunction(secretRetriableErrors)))
	cmd.AddCommand(cobras.SplitCommand(replicate.NewCmdReplicate()))
	cmd.AddCommand(cobras.SplitCommand(verify.NewCmdVerify()))
	cmd.AddCommand(cobras.SplitCommand(version.NewCmdVersion()))
	cmd.AddCommand(cobras.SplitCommand(wait.NewCmdWait()))
	cmd.AddCommand(plugins.NewCmdPlugins())
	cmd.AddCommand(vault.NewCmdVault())
	return cmd
}
