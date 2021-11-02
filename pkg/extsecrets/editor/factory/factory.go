package factory

import (
	"os"

	v1 "github.com/jenkins-x-plugins/jx-secret/pkg/apis/external/v1"
	"github.com/jenkins-x-plugins/jx-secret/pkg/apis/mapping/v1alpha1"
	"github.com/jenkins-x-plugins/jx-secret/pkg/cmd/populate"
	"github.com/jenkins-x-plugins/jx-secret/pkg/extsecrets"
	"github.com/jenkins-x-plugins/jx-secret/pkg/extsecrets/editor"
	"github.com/jenkins-x-plugins/jx-secret/pkg/vaults/vaultcli"
	"github.com/jenkins-x-plugins/secretfacade/pkg/secretstore"
	"github.com/jenkins-x-plugins/secretfacade/pkg/secretstore/factory"
	"github.com/jenkins-x-plugins/secretfacade/pkg/secretstore/kubernetessecrets"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

type secretFacadeEditor struct {
	secret        *v1.ExternalSecret
	secretManager secretstore.Interface
}

// NewEditor create a new editor using the secret store
func NewEditor(secret *v1.ExternalSecret, secretStoreManagerFactory secretstore.FactoryInterface, kubeClient kubernetes.Interface, externalVault string) (editor.Interface, error) {
	if secretStoreManagerFactory == nil {
		secretStoreManagerFactory = &factory.SecretManagerFactory{}
	}
	storeType := populate.GetSecretStore(v1alpha1.BackendType(secret.Spec.BackendType))
	if storeType == secretstore.SecretStoreTypeVault && externalVault != "true" {
		envMap, err := vaultcli.CreateVaultEnv(kubeClient)
		if err != nil {
			return nil, errors.Wrapf(err, "error creating vault env vars")
		}
		for k, v := range envMap {
			err := os.Setenv(k, v)
			if err != nil {
				return nil, errors.Wrapf(err, "failed setting env var %s for vault auth", k)
			}
		}
	}

	var secretManager secretstore.Interface
	// lets use the local kube client if available for better fake testing
	if storeType == secretstore.SecretStoreTypeKubernetes {
		secretManager = kubernetessecrets.NewKubernetesSecretManager(kubeClient)
	} else {
		var err error
		secretManager, err = secretStoreManagerFactory.NewSecretManager(storeType)
		if err != nil {
			return nil, errors.Wrapf(err, "error creating secret manager")
		}
	}
	return &secretFacadeEditor{secret: secret, secretManager: secretManager}, nil
}

func (s *secretFacadeEditor) Write(keyProperties *editor.KeyProperties) error {
	annotations := s.secret.Spec.Template.Metadata.Annotations

	key := populate.GetSecretKey(v1alpha1.BackendType(s.secret.Spec.BackendType), s.secret.Name, keyProperties.Key)

	// handle replicate to annotation for local secrets so that we also copy the secret to other namespaces
	replicateTo := ""
	if s.secret.Annotations != nil {
		replicateTo = s.secret.Annotations[extsecrets.ReplicateToAnnotation]
	}
	if replicateTo != "" {
		annotations[extsecrets.ReplicateToAnnotation] = replicateTo
	}

	labels := s.secret.Spec.Template.Metadata.Labels
	secretType := corev1.SecretType(s.secret.Spec.Template.Type)
	sv := populate.CreateSecretValue(v1alpha1.BackendType(s.secret.Spec.BackendType), keyProperties.Properties, annotations, labels, secretType)
	err := s.secretManager.SetSecret(populate.GetExternalSecretLocation(s.secret), populate.GetSecretKey(v1alpha1.BackendType(s.secret.Spec.BackendType), s.secret.Name, key), &sv)
	if err != nil {
		return errors.Wrapf(err, "failed to save properties %s on ExternalSecret %s", keyProperties.String(), s.secret.Name)
	}
	return nil
}
