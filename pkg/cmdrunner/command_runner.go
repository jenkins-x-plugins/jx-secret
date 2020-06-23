package cmdrunner

import (
	"strings"

	"github.com/jenkins-x/jx/v2/pkg/log"
	"github.com/jenkins-x/jx/v2/pkg/util"
)

// CommandRunner represents a command runner so that it can be stubbed out for testing
type CommandRunner func(*util.Command) (string, error)

// DefaultCommandRunner default runner if none is set
func DefaultCommandRunner(c *util.Command) (string, error) {
	if c.Dir == "" {
		log.Logger().Infof("about to run: %s", util.ColorInfo(CLI(c)))
	} else {
		log.Logger().Infof("about to run: %s in dir %s", util.ColorInfo(CLI(c)), util.ColorInfo(c.Dir))
	}
	result, err := c.RunWithoutRetry()
	if result != "" {
		log.Logger().Infof(util.ColorStatus(result))
	}
	return result, err
}

// DryRunCommandRunner output the commands to be run
func DryRunCommandRunner(c *util.Command) (string, error) {
	log.Logger().Infof(CLI(c))
	return "", nil
}

// CLI returns the CLI string without the dir or env vars
func CLI(c *util.Command) string {
	var builder strings.Builder
	builder.WriteString(c.Name)
	for _, arg := range c.Args {
		builder.WriteString(" ")
		builder.WriteString(arg)
	}
	return builder.String()
}
