package cmd

import (
	"github.com/jenkins-x/jx-extsecret/pkg/cmd/edit"
	"github.com/jenkins-x/jx-extsecret/pkg/cmd/export"
	importcmd "github.com/jenkins-x/jx-extsecret/pkg/cmd/import"
	"github.com/jenkins-x/jx-extsecret/pkg/cmd/verify"
	"github.com/jenkins-x/jx-extsecret/pkg/cmd/version"
	"github.com/jenkins-x/jx-promote/pkg/common"
	"github.com/jenkins-x/jx/v2/pkg/log"
	"github.com/spf13/cobra"
)

// Main creates the new command
func Main() *cobra.Command {
	cmd := &cobra.Command{
		Use:   common.TopLevelCommand,
		Short: "External Secrets utility commands",
		Run: func(cmd *cobra.Command, args []string) {
			err := cmd.Help()
			if err != nil {
				log.Logger().Errorf(err.Error())
			}
		},
	}
	cmd.AddCommand(common.SplitCommand(edit.NewCmdEdit()))
	cmd.AddCommand(common.SplitCommand(export.NewCmdExport()))
	cmd.AddCommand(common.SplitCommand(importcmd.NewCmdImport()))
	cmd.AddCommand(common.SplitCommand(verify.NewCmdVerify()))
	cmd.AddCommand(common.SplitCommand(version.NewCmdVersion()))
	return cmd
}
