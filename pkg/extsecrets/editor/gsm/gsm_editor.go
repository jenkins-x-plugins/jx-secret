package gsm

import (
	"fmt"

	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets/editor"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
)

const (
	gcloud = "gcloud"
)

type client struct {
	commandRunner cmdrunner.CommandRunner
	kubeClient    kubernetes.Interface
	env           map[string]string
}

func NewEditor(commandRunner cmdrunner.CommandRunner, kubeClient kubernetes.Interface) (editor.Interface, error) {
	if commandRunner == nil {
		commandRunner = cmdrunner.DefaultCommandRunner
	}
	c := &client{
		commandRunner: commandRunner,
		kubeClient:    kubeClient,
	}
	err := c.initialise()
	if err != nil {
		return c, errors.Wrapf(err, "failed to setup gsm secret editor")
	}
	return c, nil
}

func (c *client) Write(properties *editor.KeyProperties) error {
	key := extsecrets.SimplifyKey("gcpSecretsManager", properties.Key)

	editor.SortPropertyValues(properties.Properties)
	args := []string{"secrets", "create", key}
	for _, pv := range properties.Properties {
		args = append(args, fmt.Sprintf("%s=%s", pv.Property, pv.Value))
	}
	cmd := &cmdrunner.Command{
		Name: gcloud,
		Args: args,
		Env:  c.env,
	}

	log.Logger().Infof("would be running this command: %s", cmd.String())

	// TODO: JR shall we use the go package rather than CLI? https://cloud.google.com/secret-manager/docs/creating-and-accessing-secrets#secretmanager-add-secret-version-go
	// _, err := c.commandRunner(cmd)
	// if err != nil {
	//	return errors.Wrapf(err, "failed to invoke command")
	// }
	return nil
}

func (c *client) initialise() error {

	log.Logger().Infof("verifying we have gcloud installed")

	// lets verify we can find the binary
	cmd := &cmdrunner.Command{
		Name: gcloud,
		Args: []string{"secrets", "--help"},
		Env:  c.env,
	}
	_, err := c.commandRunner(cmd)
	if err != nil {
		return errors.Wrapf(err, "failed to invoke the binary '%s'. Please make sure you installed '%s' and put it on your $PATH", gcloud, gcloud)
	}

	log.Logger().Infof("verifying we can connect to gsm...")

	// lets verify we can list the secrets
	cmd = &cmdrunner.Command{
		Name: gcloud,
		Args: []string{"secrets", "list"},
		Env:  c.env,
	}
	_, err = c.commandRunner(cmd)
	if err != nil {
		return errors.Wrapf(err, "failed to access gsm. command failed: %s", cmdrunner.CLI(cmd))
	}

	log.Logger().Infof("gsm is setup correctly!\n\n")

	return nil
}
