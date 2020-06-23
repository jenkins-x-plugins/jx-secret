package vault

import (
	"fmt"
	"strings"

	"github.com/jenkins-x/jx-extsecret/pkg/cmdrunner"
	"github.com/jenkins-x/jx-extsecret/pkg/extsecrets/editor"
	"github.com/jenkins-x/jx/v2/pkg/util"
	"github.com/pkg/errors"
)

type client struct {
	CommandRunner cmdrunner.CommandRunner
}

func NewEditor(commandRunner cmdrunner.CommandRunner) (editor.Interface, error) {
	if commandRunner == nil {
		commandRunner = cmdrunner.DefaultCommandRunner
	}
	return &client{CommandRunner: commandRunner}, nil
}

func (c *client) Write(properties editor.KeyProperties) error {
	key := properties.Key

	// we shouldn't pass in secret/data/foo when using the CLI tool
	if strings.HasPrefix(key, "secret/data/") {
		key = "secret/" + strings.TrimPrefix(key, "secret/data/")
	}

	args := []string{"kv", "put", key}
	for _, pv := range properties.Properties {
		args = append(args, fmt.Sprintf("%s=%s", pv.Property, pv.Value))
	}
	cmd := &util.Command{
		Name: "vault",
		Args: args,
	}
	_, err := c.CommandRunner(cmd)
	if err != nil {
		return errors.Wrapf(err, "failed to invoke command")
	}
	return nil
}
