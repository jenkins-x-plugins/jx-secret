package cmd

import (
	"github.com/jenkins-x/jx-extsecret/pkg/cmd/edit"
	"github.com/jenkins-x/jx-extsecret/pkg/cmd/export"
	importcmd "github.com/jenkins-x/jx-extsecret/pkg/cmd/import"
	"github.com/jenkins-x/jx-extsecret/pkg/cmd/verify"
	"github.com/jenkins-x/jx-extsecret/pkg/cmd/version"
	"github.com/jenkins-x/jx-extsecret/pkg/root"
	"github.com/jenkins-x/jx-helpers/pkg/cobras"
	"github.com/jenkins-x/jx/v2/pkg/log"
	"github.com/spf13/cobra"
)

// Main creates the new command
func Main() *cobra.Command {
	cmd := &cobra.Command{
		Use:   root.TopLevelCommand,
		Short: "External Secrets utility commands",
		Run: func(cmd *cobra.Command, args []string) {
			err := cmd.Help()
			if err != nil {
				log.Logger().Errorf(err.Error())
			}
		},
	}
	cmd.AddCommand(cobras.SplitCommand(edit.NewCmdEdit()))
	cmd.AddCommand(cobras.SplitCommand(export.NewCmdExport()))
	cmd.AddCommand(cobras.SplitCommand(importcmd.NewCmdImport()))
	cmd.AddCommand(cobras.SplitCommand(verify.NewCmdVerify()))
	cmd.AddCommand(cobras.SplitCommand(version.NewCmdVersion()))
	return cmd
}
