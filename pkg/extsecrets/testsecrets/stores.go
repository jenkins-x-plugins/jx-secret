package testsecrets

import (
	"testing"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// DefaultStoreName is the ClusterSecretStore the test fixtures reference from
// their secretStoreRef.
const DefaultStoreName = "jx-secret-store"

// ClusterSecretStore builds a ClusterSecretStore for the fake dynamic client.
// ESO keeps the backend configuration on the store rather than the
// ExternalSecret, so a test that populates or edits secrets needs the store its
// fixtures point at to exist.
func ClusterSecretStore(t *testing.T, name string, provider *esv1.SecretStoreProvider) runtime.Object {
	store := &esv1.ClusterSecretStore{
		TypeMeta: metav1.TypeMeta{
			APIVersion: esv1.SchemeGroupVersion.String(),
			Kind:       esv1.ClusterSecretStoreKind,
		},
		// cluster-scoped, so deliberately no namespace: storeClient looks it up
		// without one and the fake client matches on that
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

// GCPSMProvider describes Google Secret Manager in the given project.
func GCPSMProvider(projectID string) *esv1.SecretStoreProvider {
	return &esv1.SecretStoreProvider{GCPSM: &esv1.GCPSMProvider{ProjectID: projectID}}
}

// AzureKVProvider describes the Azure key vault of the given name.
func AzureKVProvider(vaultName string) *esv1.SecretStoreProvider {
	return &esv1.SecretStoreProvider{AzureKV: &esv1.AzureKVProvider{
		VaultURL: new("https://" + vaultName + ".vault.azure.net"),
	}}
}

// KubernetesProvider describes local secrets in the given namespace.
func KubernetesProvider(namespace string) *esv1.SecretStoreProvider {
	return &esv1.SecretStoreProvider{Kubernetes: &esv1.KubernetesProvider{RemoteNamespace: namespace}}
}
