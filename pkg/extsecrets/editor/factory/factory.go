package factory

import (
	"os"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
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
	secret        *esv1.ExternalSecret
	secretManager secretstore.Interface
	backend       *extsecrets.Backend
}

// NewEditor create a new editor writing to the given resolved backend.
func NewEditor(secret *esv1.ExternalSecret, backend *extsecrets.Backend, secretStoreManagerFactory secretstore.FactoryInterface, kubeClient kubernetes.Interface, externalVault string) (editor.Interface, error) {
	if secretStoreManagerFactory == nil {
		secretStoreManagerFactory = &factory.SecretManagerFactory{}
	}
	storeType := populate.GetSecretStore(backend.Type)
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
	return &secretFacadeEditor{secret: secret, secretManager: secretManager, backend: backend}, nil
}

func (s *secretFacadeEditor) Write(keyProperties *editor.KeyProperties) error {
	var annotations map[string]string
	var labels map[string]string
	secretType := corev1.SecretTypeOpaque
	if s.secret.Spec.Target.Template != nil {
		annotations = s.secret.Spec.Target.Template.Metadata.Annotations
		labels = s.secret.Spec.Target.Template.Metadata.Labels
		if s.secret.Spec.Target.Template.Type != "" {
			secretType = s.secret.Spec.Target.Template.Type
		}
	}
	backend := s.backend.Type
	key := populate.GetSecretKey(backend, s.secret.Name, keyProperties.Key)

	// handle replicate to annotation for local secrets so that we also copy the secret to other namespaces
	replicateTo := ""
	if s.secret.Annotations != nil {
		replicateTo = s.secret.Annotations[extsecrets.ReplicateToAnnotation]
	}
	if replicateTo != "" {
		if annotations == nil {
			annotations = map[string]string{}
		}
		annotations[extsecrets.ReplicateToAnnotation] = replicateTo
	}

	sv := populate.CreateSecretValue(backend, keyProperties.Properties, annotations, labels, secretType)
	err := s.secretManager.SetSecret(s.backend.Location, populate.GetSecretKey(backend, s.secret.Name, key), &sv)
	if err != nil {
		return errors.Wrapf(err, "failed to save properties %s on ExternalSecret %s", keyProperties.String(), s.secret.Name)
	}
	return nil
}
