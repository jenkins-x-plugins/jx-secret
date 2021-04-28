package gsm

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"

	"k8s.io/apimachinery/pkg/util/json"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"github.com/jenkins-x-plugins/jx-secret/pkg/extsecrets"
	"github.com/jenkins-x-plugins/jx-secret/pkg/extsecrets/editor"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/pkg/errors"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
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
	tmpDir             string
}

func NewEditor(commandRunner cmdrunner.CommandRunner, quietCommandRunner cmdrunner.CommandRunner, kubeClient kubernetes.Interface) (editor.Interface, error) {
	if commandRunner == nil {
		commandRunner = cmdrunner.DefaultCommandRunner
	}
	if quietCommandRunner == nil {
		quietCommandRunner = cmdrunner.QuietCommandRunner
	}

	tmpDir := os.Getenv("JX_SECRET_TMP_DIR")
	if tmpDir != "" {
		err := os.MkdirAll(tmpDir, files.DefaultDirWritePermissions)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create jx secret temp dir %s", tmpDir)
		}
	}
	log.Logger().Debugf("using secret temp dir %s", tmpDir)

	c := &client{
		commandRunner:      commandRunner,
		quietCommandRunner: quietCommandRunner,
		kubeClient:         kubeClient,
		tmpDir:             tmpDir,
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
	file, err := ioutil.TempFile(c.tmpDir, "jx")
	if err != nil {
		return errors.Wrap(err, "failed to create temporary directory used to write secrets to then upload to google secrets manager")
	}

	defer os.Remove(file.Name())

	// only replace properties that we are editing, so first lets get the existing values
	existingValues, err := accessSecretVersion(properties.GCPProject, properties.Key)
	if err != nil {
		return errors.Wrapf(err, "failed to access secret %s in project %s", properties.Key, properties.GCPProject)
	}

	// write properties as a key values ina  json file so we can upload to Google Secrets Manager
	err = c.writeTemporarySecretPropertiesJSON(existingValues, properties, file)
	if err != nil {
		return errors.Wrapf(err, "failed to write secret key values pairs to filename %s", file.Name())
	}

	//create a new secret version
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

func (c *client) writeTemporarySecretPropertiesJSON(existingValues map[string]string, properties *editor.KeyProperties, file *os.File) error {
	// if we only have one property and its got an empty property name lets just write the value
	if len(properties.Properties) == 1 && properties.Properties[0].Property == "" {
		_, err := file.Write([]byte(properties.Properties[0].Value))
		if err != nil {
			return errors.Wrap(err, "failed to write property value to temporary file")
		}
		return nil
	}

	// write properties as a key values ina  json file so we can upload to Google Secrets Manager
	for _, p := range properties.Properties {
		existingValues[p.Property] = p.Value
	}

	data, err := json.Marshal(existingValues)
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

func VerifyGcloudInstalled() error {

	log.Logger().Debugf("verifying we have gcloud installed")

	runner := cmdrunner.QuietCommandRunner
	// lets verify we can find the binary
	cmd := &cmdrunner.Command{
		Name: gcloud,
		Args: []string{"secrets", "--help"},
	}
	_, err := runner(cmd)
	if err != nil {
		return errors.Wrapf(err, "failed to invoke the binary '%s'. Please make sure you installed '%s' and put it on your $PATH", gcloud, gcloud)
	}

	log.Logger().Debugf("verifying we can connect to gsm...")

	// lets verify we can list the secrets
	cmd = &cmdrunner.Command{
		Name: gcloud,
		Args: []string{"secrets", "list", "--help"},
	}
	_, err = runner(cmd)
	if err != nil {
		return errors.Wrapf(err, "failed to access gsm. command failed: %s", cmdrunner.CLI(cmd))
	}

	log.Logger().Debugf("gsm is setup correctly!\n\n")

	return nil
}

func accessSecretVersion(projectID, key string) (map[string]string, error) {

	name := fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectID, key)

	// Create the client.
	ctx := context.Background()
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create secretmanager client: %v", err)
	}

	// Build the request.
	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: name,
	}

	// Call the API.
	result, err := client.AccessSecretVersion(ctx, req)
	if err != nil && !strings.Contains(err.Error(), "not found or has no versions") {
		return nil, errors.Wrapf(err, "failed to access secret version %s", name)
	}

	m := make(map[string]string)
	if result != nil && len(result.Payload.Data) > 0 && strings.TrimSpace(string(result.Payload.Data)) != "" {
		err = json.Unmarshal(result.Payload.Data, &m)
		if err != nil {
			return m, errors.Wrapf(err, "failed to parse JSON '%s' from GSM secret version", string(result.Payload.Data))
		}
	}
	return m, nil
}
