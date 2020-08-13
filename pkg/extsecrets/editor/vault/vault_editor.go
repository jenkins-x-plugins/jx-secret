package vault

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/pkg/files"
	"github.com/jenkins-x/jx-helpers/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets/editor"
	"github.com/jenkins-x/jx-secret/pkg/plugins"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		commandRunner = cmdrunner.DefaultCommandRunner
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

// VaultCommandRunner do not output the
func VaultCommandRunner(c *cmdrunner.Command) (string, error) {
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
	result := []string{}
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
	c.vaultBin = os.Getenv("VAULT_BIN")
	if c.vaultBin == "" {
		var err error
		c.vaultBin, err = plugins.GetVaultBinary(plugins.VaultVersion)
		if err != nil {
			return errors.Wrapf(err, "failed to find version %s of the vault plugin binary", plugins.VaultVersion)
		}
	}

	log.Logger().Infof("verifying we have vault installed")

	// lets verify we can find the binary
	cmd := &cmdrunner.Command{
		Name: c.vaultBin,
		Args: []string{"version"},
		Env:  c.env,
	}
	_, err := c.commandRunner(cmd)
	if err != nil {
		return errors.Wrapf(err, "failed to invoke the binary %s. Please make sure you installed 'vault' and put it on your $PATH", c.vaultBin)
	}

	addr := os.Getenv("VAULT_ADDR")
	if addr == "" {
		addr = "https://127.0.0.1:8200"
	}
	ns := os.Getenv("VAULT_NAMESPACE")
	if ns == "" {
		ns = "vault-infra"
	}
	token := os.Getenv("VAULT_TOKEN")
	if token == "" {
		token, err = getSecretKey(c.kubeClient, ns, "vault-unseal-keys", "vault-root")
		if err != nil {
			return err
		}
	}

	caCertFile := os.Getenv("VAULT_CACERT")
	if caCertFile == "" {
		tmpDir, err := ioutil.TempDir("", "jx-secret-vault-") //nolint:govet
		if err != nil {
			return errors.Wrapf(err, "failed to create temp dir")
		}

		caCert, err := getSecretKey(c.kubeClient, ns, "vault-tls", "ca.crt")
		if err != nil {
			return err
		}
		caCertFile = filepath.Join(tmpDir, "vault-ca.crt")
		err = ioutil.WriteFile(caCertFile, []byte(caCert), files.DefaultFileWritePermissions)
		if err != nil {
			return errors.Wrapf(err, "failed to save CA Cert file %s", caCertFile)
		}
	}
	c.env = map[string]string{
		"VAULT_ADDR":   addr,
		"VAULT_TOKEN":  token,
		"VAULT_CACERT": caCertFile,
	}

	log.Logger().Infof("verifying we can connect to vault...")

	// lets verify we can list the secrets
	cmd = &cmdrunner.Command{
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

func getSecretKey(kubeClient kubernetes.Interface, ns, secretName, key string) (string, error) {
	secret, err := kubeClient.CoreV1().Secrets(ns).Get(secretName, metav1.GetOptions{})
	if err != nil && apierrors.IsNotFound(err) {
		err = nil
	}
	if err != nil {
		return "", errors.Wrapf(err, "failed to find secret %s in namespace %s", secretName, ns)
	}
	if secret == nil || secret.Data == nil {
		return "", errors.Errorf("no data for secret %s in namespace %s", secretName, ns)
	}
	value := secret.Data[key]
	if len(value) == 0 {
		return "", errors.Errorf("no '%s' entry for secret %s in namespace %s", key, secretName, ns)
	}
	return string(value), nil
}
