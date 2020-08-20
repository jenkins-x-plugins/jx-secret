package vault

import (
	"fmt"
	"strings"

	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets/editor"
	"github.com/jenkins-x/jx-secret/pkg/vaults/vaultcli"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
)

type client struct {
	commandRunner cmdrunner.CommandRunner
	kubeClient    kubernetes.Interface
	env           map[string]string
	vaultBin      string
}

func NewEditor(commandRunner cmdrunner.CommandRunner, kubeClient kubernetes.Interface) (editor.Interface, error) {
	if commandRunner == nil {
		commandRunner = MaskedCommandRunner
	}
	c := &client{
		commandRunner: commandRunner,
		kubeClient:    kubeClient,
	}
	err := c.initialise()
	if err != nil {
		return c, errors.Wrapf(err, "failed to setup vault secret editor")
	}
	return c, nil
}

// MaskedCommandRunner mask the command line arguments when logging
func MaskedCommandRunner(c *cmdrunner.Command) (string, error) {
	args := MastSecretArgs(c.Args)
	log.Logger().Infof("about to run: %s %s", termcolor.ColorInfo(c.Name), termcolor.ColorInfo(strings.Join(args, " ")))

	result, err := c.RunWithoutRetry()
	if result != "" {
		log.Logger().Infof(termcolor.ColorStatus(result))
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

func (c *client) Write(properties *editor.KeyProperties) error {
	key := extsecrets.SimplifyKey("vault", properties.Key)

	editor.SortPropertyValues(properties.Properties)
	args := []string{"kv", "put", key}
	for _, pv := range properties.Properties {
		args = append(args, fmt.Sprintf("%s=%s", pv.Property, pv.Value))
	}
	cmd := &cmdrunner.Command{
		Name: c.vaultBin,
		Args: args,
		Env:  c.env,
	}
	_, err := c.commandRunner(cmd)
	if err != nil {
		return errors.Wrapf(err, "failed to invoke command")
	}
	return nil
}

func (c *client) initialise() error {
	var err error
	c.vaultBin, err = vaultcli.VerifyVaultBinary(c.commandRunner, c.env)
	if err != nil {
		return errors.Wrapf(err, "failed to validate vault binary")
	}

	c.env, err = vaultcli.CreateVaultEnv(c.kubeClient)
	if err != nil {
		return errors.Wrapf(err, "failed to setup the vault environment")
	}

	log.Logger().Infof("verifying we can connect to vault...")

	// lets verify we can list the secrets
	cmd := &cmdrunner.Command{
		Name: c.vaultBin,
		Args: []string{"kv", "list", "secret"},
		Env:  c.env,
	}
	_, err = c.commandRunner(cmd)
	if err != nil {
		return errors.Wrapf(err, "failed to access vault. are you sure you are running the 'jx-secret vault portforward' command? command failed: %s", cmdrunner.CLI(cmd))
	}

	log.Logger().Infof("vault is setup correctly!\n\n")

	return nil
}
