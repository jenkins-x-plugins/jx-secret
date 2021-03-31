package vaultcli

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/jenkins-x-plugins/jx-secret/pkg/plugins"
	"github.com/jenkins-x-plugins/jx-secret/pkg/vaults"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// VerifyVaultBinary verifies the vault binary
func VerifyVaultBinary(commandRunner cmdrunner.CommandRunner, env map[string]string) (string, error) {
	vaultBin := os.Getenv("VAULT_BIN")
	if vaultBin == "" {
		var err error
		vaultBin, err = plugins.GetVaultBinary(plugins.VaultVersion)
		if err != nil {
			return "", errors.Wrapf(err, "failed to find version %s of the vault plugin binary", plugins.VaultVersion)
		}
	}

	log.Logger().Infof("verifying we have vault installed")

	// lets verify we can find the binary
	cmd := &cmdrunner.Command{
		Name: vaultBin,
		Args: []string{"version"},
		Env:  env,
	}
	_, err := commandRunner(cmd)
	if err != nil {
		return "", errors.Wrapf(err, "failed to invoke the binary %s. Please make sure you installed 'vault' and put it on your $PATH", vaultBin)
	}
	return vaultBin, nil
}

// CreateVaultEnv creates the vault env vars
func CreateVaultEnv(kubeClient kubernetes.Interface) (map[string]string, error) {
	addr := os.Getenv("VAULT_ADDR")
	if addr == "" {
		addr = "https://127.0.0.1:8200"
	}
	ns := os.Getenv("VAULT_NAMESPACE")
	if ns == "" {
		ns = vaults.DefaultVaultNamespace
	}
	token := os.Getenv("VAULT_TOKEN")
	var err error
	if token == "" {
		token, err = getSecretKey(kubeClient, ns, "vault-unseal-keys", "vault-root")
		if err != nil {
			return nil, err
		}
	}

	caCertFile := os.Getenv("VAULT_CACERT")
	if caCertFile == "" {
		tmpDir, err := ioutil.TempDir("", "jx-secret-vault-") //nolint:govet
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create temp dir")
		}

		caCert, err := getSecretKey(kubeClient, ns, "vault-tls", "ca.crt")
		if err != nil {
			return nil, err
		}
		caCertFile = filepath.Join(tmpDir, "vault-ca.crt")
		err = ioutil.WriteFile(caCertFile, []byte(caCert), files.DefaultFileWritePermissions)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to save CA Cert file %s", caCertFile)
		}
	}
	env := map[string]string{
		"VAULT_ADDR":   addr,
		"VAULT_TOKEN":  token,
		"VAULT_CACERT": caCertFile,
	}
	return env, nil
}

func getSecretKey(kubeClient kubernetes.Interface, ns, secretName, key string) (string, error) {
	secret, err := kubeClient.CoreV1().Secrets(ns).Get(context.TODO(), secretName, metav1.GetOptions{})
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
