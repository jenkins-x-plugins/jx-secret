package factory

import (
	v1 "github.com/jenkins-x-plugins/jx-secret/pkg/apis/external/v1"
	"github.com/jenkins-x-plugins/jx-secret/pkg/apis/mapping/v1alpha1"
	"github.com/jenkins-x-plugins/jx-secret/pkg/extsecrets/editor"
	"github.com/jenkins-x-plugins/jx-secret/pkg/extsecrets/editor/azure"
	"github.com/jenkins-x-plugins/jx-secret/pkg/extsecrets/editor/gsm"
	"github.com/jenkins-x-plugins/jx-secret/pkg/extsecrets/editor/local"
	"github.com/jenkins-x-plugins/jx-secret/pkg/extsecrets/editor/vault"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
)

func NewEditor(cache map[string]editor.Interface, secret *v1.ExternalSecret, commandRunner cmdrunner.CommandRunner, quietCommandRunner cmdrunner.CommandRunner, client kubernetes.Interface) (editor.Interface, error) {
	backendType := v1alpha1.BackendType(secret.Spec.BackendType)
	keyVaultName := secret.Spec.KeyVaultName
	switch backendType {
	case v1alpha1.BackendTypeLocal:
		return local.NewEditor(client, secret)
	case v1alpha1.BackendTypeVault:
		// lets cache vault editors
		cached := cache["vault"]
		var err error
		if cached == nil {
			cached, err = vault.NewEditor(commandRunner, quietCommandRunner, client)
			if err != nil {
				return cached, errors.Wrapf(err, "failed to create vault editor")
			}
			cache["vault"] = cached
		}
		return cached, nil
	case v1alpha1.BackendTypeGSM:
		return gsm.NewEditor(commandRunner, quietCommandRunner, client)
	case v1alpha1.BackendTypeAzure:
		return azure.NewEditor(keyVaultName, azure.KeyVaultClient{})
	default:
		return nil, errors.Errorf("unsupported ExternalSecret back end %s", backendType)
	}
}
