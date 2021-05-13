package main

import (
	"github.com/jenkins-x-plugins/jx-secret/pkg/cmd/postrender"
	"os"
)

// Run runs the command, if args are not nil they will be set on the command
func main() {
	cmd, _ := postrender.NewCmdPostrender()

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
