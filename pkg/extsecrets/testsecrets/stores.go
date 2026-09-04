package testsecrets

import (
	"testing"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// DefaultStoreName is the store the test fixtures point their secretStoreRef at.
const DefaultStoreName = "jx-secret-store"

// ClusterSecretStore builds a store for the fake dynamic client. Populate and edit
// resolve the backend from it, so their fixtures need it to exist.
func ClusterSecretStore(t *testing.T, name string, provider *esv1.SecretStoreProvider) runtime.Object {
	store := &esv1.ClusterSecretStore{
		TypeMeta: metav1.TypeMeta{
			APIVersion: esv1.SchemeGroupVersion.String(),
			Kind:       esv1.ClusterSecretStoreKind,
		},
		// cluster-scoped: storeClient looks it up without a namespace
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec:       esv1.SecretStoreSpec{Provider: provider},
	}

	content, err := runtime.DefaultUnstructuredConverter.ToUnstructured(store)
	require.NoError(t, err, "failed to convert ClusterSecretStore %s to unstructured", name)
	return &unstructured.Unstructured{Object: content}
}

// VaultProvider describes a KV v2 mount named "secret" on the given server.
func VaultProvider(server string) *esv1.SecretStoreProvider {
	return &esv1.SecretStoreProvider{Vault: &esv1.VaultProvider{
		Server:  server,
		Path:    new("secret"),
		Version: esv1.VaultKVStoreV2,
	}}
}

func GCPSMProvider(projectID string) *esv1.SecretStoreProvider {
	return &esv1.SecretStoreProvider{GCPSM: &esv1.GCPSMProvider{ProjectID: projectID}}
}

// AzureKVProvider synthesises the vault URL ESO expects from a bare vault name.
func AzureKVProvider(vaultName string) *esv1.SecretStoreProvider {
	return &esv1.SecretStoreProvider{AzureKV: &esv1.AzureKVProvider{
		VaultURL: new("https://" + vaultName + ".vault.azure.net"),
	}}
}

func KubernetesProvider(namespace string) *esv1.SecretStoreProvider {
	return &esv1.SecretStoreProvider{Kubernetes: &esv1.KubernetesProvider{RemoteNamespace: namespace}}
}
