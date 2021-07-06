package vaultcli

import (
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
)

type client struct {
	commandRunner      cmdrunner.CommandRunner
	quietCommandRunner cmdrunner.CommandRunner
	kubeClient         kubernetes.Interface
	env                map[string]string
	vaultBin           string
}

func WaitForVault(commandRunner cmdrunner.CommandRunner, quietCommandRunner cmdrunner.CommandRunner, kubeClient kubernetes.Interface) error {
	if commandRunner == nil {
		commandRunner = MaskedCommandRunner
	}
	if quietCommandRunner == nil {
		quietCommandRunner = commandRunner
	}
	c := &client{
		commandRunner:      commandRunner,
		quietCommandRunner: quietCommandRunner,
		kubeClient:         kubeClient,
	}
	err := c.initialise()
	if err != nil {
		return errors.Wrapf(err, "failed to wait for vault to be available")
	}
	return nil
}

func (c *client) initialise() error {
	var err error
	c.vaultBin, err = VerifyVaultBinary(c.commandRunner, c.env)
	if err != nil {
		return errors.Wrapf(err, "failed to validate vault binary")
	}

	c.env, err = CreateVaultEnv(c.kubeClient)
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
