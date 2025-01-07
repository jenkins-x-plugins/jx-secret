package vaultcli

import (
	"fmt"
	"strings"

	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
)

// MaskedCommandRunner mask the command line arguments when logging
func MaskedCommandRunner(c *cmdrunner.Command) (string, error) {
	args := MastSecretArgs(c.Args)
	log.Logger().Infof("about to run: %s %s", termcolor.ColorInfo(c.Name), termcolor.ColorInfo(strings.Join(args, " ")))

	result, err := c.RunWithoutRetry()
	if result != "" {
		log.Logger().Info(termcolor.ColorStatus(result))
	}
	return result, err
}

// MastSecretArgs lets mask any passwords/tokens in the arguments passed into the vault CLI
// e.g. for the arguments: :
// kv put secret/jx/pipelineUser token=dummyPipelineToken
func MastSecretArgs(args []string) []string {
	if len(args) < 3 {
		return args
	}
	var result []string
	result = append(result, args...)
	for i := 2; i < len(result); i++ {
		values := strings.SplitN(result[i], "=", 2)
		if len(values) == 2 {
			result[i] = fmt.Sprintf("%s=****", values[0])
		}
	}
	return result
}
