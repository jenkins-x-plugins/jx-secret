package gsm

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/jenkins-x/jx-logging/v3/pkg/log"

	"k8s.io/apimachinery/pkg/util/json"

	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets/editor"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
)

const (
	gcloud = "gcloud"
)

type client struct {
	commandRunner      cmdrunner.CommandRunner
	quietCommandRunner cmdrunner.CommandRunner
	kubeClient         kubernetes.Interface
	env                map[string]string
}

func NewEditor(commandRunner cmdrunner.CommandRunner, kubeClient kubernetes.Interface) (editor.Interface, error) {
	if commandRunner == nil {
		commandRunner = cmdrunner.DefaultCommandRunner
	}

	c := &client{
		commandRunner:      commandRunner,
		kubeClient:         kubeClient,
		quietCommandRunner: cmdrunner.QuietCommandRunner,
	}
	err := c.initialise()
	if err != nil {
		return c, errors.Wrapf(err, "failed to setup gsm secret editor")
	}
	return c, nil
}

func (c *client) Write(properties *editor.KeyProperties) error {
	key := extsecrets.SimplifyKey("gcpSecretsManager", properties.Key)

	if len(properties.Properties) == 0 {
		return fmt.Errorf("creating an inline secret in Google Secret Manager with no property is not yet supported, secret %s", key)
	}

	// check secret is created
	err := c.ensureSecretExists(key, properties.GCPProject)
	if err != nil {
		return errors.Wrapf(err, "failed to ensure secret key %s exists in project %s", key, properties.GCPProject)
	}

	editor.SortPropertyValues(properties.Properties)

	// create a temporary file used to upload secret values to Google Secrets Manager
	file, err := ioutil.TempFile("", "jx")
	if err != nil {
		return errors.Wrap(err, "failed to create temporary directory used to write secrets to then upload to google secrets manager")
	}
	defer os.Remove(file.Name())

	// write properties as a key values ina  json file so we can upload to Google Secrets Manager
	err = c.writeTemporarySecretPropertiesJSON(properties, file)
	if err != nil {
		return errors.Wrapf(err, "failed to write secret key values pairs to filename %s", file.Name())
	}

	// create a new secret version
	args := []string{"secrets", "versions", "add", key, "--project", properties.GCPProject, "--data-file", file.Name()}
	cmd := &cmdrunner.Command{
		Name: gcloud,
		Args: args,
		Env:  c.env,
	}

	_, err = c.commandRunner(cmd)
	if err != nil {
		return errors.Wrapf(err, "failed to create a version of secret %s", key)
	}
	return nil
}

func (c *client) writeTemporarySecretPropertiesJSON(properties *editor.KeyProperties, file *os.File) error {
	// if we only have one property and its got an empty property name lets just write the value
	if len(properties.Properties) == 1 && properties.Properties[0].Property == "" {
		_, err := file.Write([]byte(properties.Properties[0].Value))
		if err != nil {
			return errors.Wrap(err, "failed to write property value to temporary file")
		}
		return nil
	}

	// write properties as a key values ina  json file so we can upload to Google Secrets Manager
	values := map[string]string{}
	for _, p := range properties.Properties {
		values[p.Property] = p.Value
	}

	data, err := json.Marshal(values)
	if err != nil {
		return errors.Wrap(err, "failed to marshall secrets used to upload to google secrets manager")
	}

	_, err = file.Write(data)
	if err != nil {
		return errors.Wrap(err, "failed to write secrets to then upload to google secrets manager")
	}
	return nil
}

func (c *client) ensureSecretExists(key, projectID string) error {
	args := []string{"secrets", "describe", key, "--project", projectID}
	cmd := &cmdrunner.Command{
		Name: gcloud,
		Args: args,
		Env:  c.env,
	}
	_, err := c.quietCommandRunner(cmd)
	if err != nil {
		if strings.Contains(err.Error(), "NOT_FOUND") {
			args := []string{"secrets", "create", key, "--project", projectID, "--replication-policy", "automatic"}

			cmd := &cmdrunner.Command{
				Name: gcloud,
				Args: args,
				Env:  c.env,
			}
			_, err = c.commandRunner(cmd)
			if err != nil {
				return errors.Wrapf(err, "failed to create secret %s in project %s", key, projectID)
			}
		} else {
			return errors.Wrapf(err, "failed to describe secret %s in project %s", key, projectID)
		}
	}
	return nil
}

func (c *client) initialise() error {

	log.Logger().Debugf("verifying we have gcloud installed")

	// lets verify we can find the binary
	cmd := &cmdrunner.Command{
		Name: gcloud,
		Args: []string{"secrets", "--help"},
		Env:  c.env,
	}
	_, err := c.quietCommandRunner(cmd)
	if err != nil {
		return errors.Wrapf(err, "failed to invoke the binary '%s'. Please make sure you installed '%s' and put it on your $PATH", gcloud, gcloud)
	}

	log.Logger().Debugf("verifying we can connect to gsm...")

	// lets verify we can list the secrets
	cmd = &cmdrunner.Command{
		Name: gcloud,
		Args: []string{"secrets", "list", "--help"},
		Env:  c.env,
	}
	_, err = c.quietCommandRunner(cmd)
	if err != nil {
		return errors.Wrapf(err, "failed to access gsm. command failed: %s", cmdrunner.CLI(cmd))
	}

	log.Logger().Debugf("gsm is setup correctly!\n\n")

	return nil
}
