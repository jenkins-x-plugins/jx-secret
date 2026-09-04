package extsecrets_test

import (
	"testing"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/jenkins-x-plugins/jx-secret/pkg/apis/mapping/v1alpha1"
	"github.com/jenkins-x-plugins/jx-secret/pkg/extsecrets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// fakeStores counts lookups so the caching test can assert on them.
type fakeStores struct {
	stores map[string]esv1.GenericStore
	calls  int
	err    error
}

func (f *fakeStores) GetStore(kind, name, _ string) (esv1.GenericStore, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	store, ok := f.stores[kind+"/"+name]
	if !ok {
		return nil, assert.AnError
	}
	return store, nil
}

// storesWith serves one store named "store", which is what externalSecret refers to.
func storesWith(provider *esv1.SecretStoreProvider) *fakeStores {
	store := &esv1.ClusterSecretStore{
		ObjectMeta: metav1.ObjectMeta{Name: "store"},
		Spec:       esv1.SecretStoreSpec{Provider: provider},
	}
	return &fakeStores{stores: map[string]esv1.GenericStore{
		esv1.ClusterSecretStoreKind + "/store": store,
	}}
}

func externalSecret(name, namespace, storeName string) *esv1.ExternalSecret {
	return &esv1.ExternalSecret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: esv1.ExternalSecretSpec{
			SecretStoreRef: esv1.SecretStoreRef{Name: storeName, Kind: esv1.ClusterSecretStoreKind},
		},
	}
}

func strPtr(s string) *string { return &s }

func TestResolveFromStore(t *testing.T) {
	testCases := []struct {
		name         string
		provider     *esv1.SecretStoreProvider
		wantBackend  v1alpha1.BackendType
		wantLocation string
	}{
		{
			name:         "vault",
			provider:     &esv1.SecretStoreProvider{Vault: &esv1.VaultProvider{Server: "https://vault:8200"}},
			wantBackend:  v1alpha1.BackendTypeVault,
			wantLocation: "https://vault:8200",
		},
		{
			name:         "gcp secrets manager",
			provider:     &esv1.SecretStoreProvider{GCPSM: &esv1.GCPSMProvider{ProjectID: "my-project"}},
			wantBackend:  v1alpha1.BackendTypeGSM,
			wantLocation: "my-project",
		},
		{
			name:         "azure key vault reduces the url to the vault name",
			provider:     &esv1.SecretStoreProvider{AzureKV: &esv1.AzureKVProvider{VaultURL: strPtr("https://my-vault.vault.azure.net")}},
			wantBackend:  v1alpha1.BackendTypeAzure,
			wantLocation: "my-vault",
		},
		{
			name:         "aws secrets manager",
			provider:     &esv1.SecretStoreProvider{AWS: &esv1.AWSProvider{Service: esv1.AWSServiceSecretsManager, Region: "eu-west-2"}},
			wantBackend:  v1alpha1.BackendTypeAWSSecretsManager,
			wantLocation: "eu-west-2",
		},
		{
			name:         "aws parameter store",
			provider:     &esv1.SecretStoreProvider{AWS: &esv1.AWSProvider{Service: esv1.AWSServiceParameterStore, Region: "us-east-1"}},
			wantBackend:  v1alpha1.BackendTypeAWSParameterStore,
			wantLocation: "us-east-1",
		},
		{
			name:         "kubernetes uses the remote namespace",
			provider:     &esv1.SecretStoreProvider{Kubernetes: &esv1.KubernetesProvider{RemoteNamespace: "secret-infra"}},
			wantBackend:  v1alpha1.BackendTypeLocal,
			wantLocation: "secret-infra",
		},
		{
			name:         "kubernetes falls back to the ExternalSecret namespace",
			provider:     &esv1.SecretStoreProvider{Kubernetes: &esv1.KubernetesProvider{}},
			wantBackend:  v1alpha1.BackendTypeLocal,
			wantLocation: "jx",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := &extsecrets.BackendResolver{Stores: storesWith(tc.provider)}

			b, err := r.Resolve(externalSecret("my-secret", "jx", "store"))
			require.NoError(t, err)
			assert.Equal(t, tc.wantBackend, b.Type, "backend type")
			assert.Equal(t, tc.wantLocation, b.Location, "location")
		})
	}
}

func TestResolveFailsWhenStoreUnreadable(t *testing.T) {
	r := &extsecrets.BackendResolver{Stores: &fakeStores{err: assert.AnError}}

	_, err := r.Resolve(externalSecret("my-secret", "jx", "jx-secret-store"))
	require.Error(t, err, "must not guess at the backend")
	assert.Contains(t, err.Error(), "jx-secret-store", "the error should name the store")
	assert.Contains(t, err.Error(), "jx/my-secret", "the error should name the ExternalSecret")
}

func TestResolveFailsWhenStoreProviderUnsupported(t *testing.T) {
	r := &extsecrets.BackendResolver{
		Stores: storesWith(&esv1.SecretStoreProvider{Fake: &esv1.FakeProvider{}}),
	}

	_, err := r.Resolve(externalSecret("my-secret", "jx", "store"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider jx-secret cannot write to")
}

func TestResolveFailsWithoutStoreRef(t *testing.T) {
	r := &extsecrets.BackendResolver{
		Stores: storesWith(&esv1.SecretStoreProvider{GCPSM: &esv1.GCPSMProvider{ProjectID: "p"}}),
	}

	_, err := r.Resolve(externalSecret("my-secret", "jx", ""))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "references no SecretStore")
}

func TestRemoteKeyPath(t *testing.T) {
	testCases := []struct {
		name     string
		provider *esv1.SecretStoreProvider
		key      string
		want     string
	}{
		{
			name:     "kv v2 gains the data segment",
			provider: &esv1.SecretStoreProvider{Vault: &esv1.VaultProvider{Version: esv1.VaultKVStoreV2}},
			key:      "jx/pipelineUser",
			want:     "secret/data/jx/pipelineUser",
		},
		{
			name:     "an unset version is treated as kv v2, as ESO does",
			provider: &esv1.SecretStoreProvider{Vault: &esv1.VaultProvider{}},
			key:      "jx/pipelineUser",
			want:     "secret/data/jx/pipelineUser",
		},
		{
			name:     "kv v1 has no data segment",
			provider: &esv1.SecretStoreProvider{Vault: &esv1.VaultProvider{Version: esv1.VaultKVStoreV1}},
			key:      "jx/pipelineUser",
			want:     "secret/jx/pipelineUser",
		},
		{
			name:     "a custom mount is honoured",
			provider: &esv1.SecretStoreProvider{Vault: &esv1.VaultProvider{Path: strPtr("jx-kv"), Version: esv1.VaultKVStoreV2}},
			key:      "jx/pipelineUser",
			want:     "jx-kv/data/jx/pipelineUser",
		},
		{
			name:     "a key that already carries the mount is left alone",
			provider: &esv1.SecretStoreProvider{Vault: &esv1.VaultProvider{Version: esv1.VaultKVStoreV2}},
			key:      "secret/data/jx/pipelineUser",
			want:     "secret/data/jx/pipelineUser",
		},
		{
			name:     "non vault backends pass the key through",
			provider: &esv1.SecretStoreProvider{GCPSM: &esv1.GCPSMProvider{ProjectID: "p"}},
			key:      "jx-pipeline-user",
			want:     "jx-pipeline-user",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := &extsecrets.BackendResolver{Stores: storesWith(tc.provider)}
			b, err := r.Resolve(externalSecret("s", "jx", "store"))
			require.NoError(t, err)
			assert.Equal(t, tc.want, b.RemoteKeyPath(tc.key))
		})
	}
}

func TestResolveCachesPerStore(t *testing.T) {
	stores := storesWith(&esv1.SecretStoreProvider{GCPSM: &esv1.GCPSMProvider{ProjectID: "p"}})
	r := &extsecrets.BackendResolver{Stores: stores}

	for range 5 {
		b, err := r.Resolve(externalSecret("secret", "jx", "store"))
		require.NoError(t, err)
		require.Equal(t, v1alpha1.BackendTypeGSM, b.Type)
	}
	assert.Equal(t, 1, stores.calls, "the store should be read once and cached")
}

// a store with no remoteNamespace falls back to each ExternalSecret's own namespace
func TestResolveLocalNamespaceIsNotCached(t *testing.T) {
	r := &extsecrets.BackendResolver{
		Stores: storesWith(&esv1.SecretStoreProvider{Kubernetes: &esv1.KubernetesProvider{}}),
	}

	jx, err := r.Resolve(externalSecret("s", "jx", "store"))
	require.NoError(t, err)
	assert.Equal(t, "jx", jx.Location)

	staging, err := r.Resolve(externalSecret("s", "jx-staging", "store"))
	require.NoError(t, err)
	assert.Equal(t, "jx-staging", staging.Location)
}

// VAULT_ADDR only appears once the vault port-forward is up, mid-run
func TestResolveVaultAddressIsReadOnAccess(t *testing.T) {
	r := &extsecrets.BackendResolver{
		Stores: storesWith(&esv1.SecretStoreProvider{Vault: &esv1.VaultProvider{}}),
	}
	es := externalSecret("s", "jx", "store")

	before, err := r.Resolve(es)
	require.NoError(t, err)
	require.Empty(t, before.Location)

	t.Setenv("VAULT_ADDR", "https://127.0.0.1:8200")
	after, err := r.Resolve(es)
	require.NoError(t, err)
	assert.Equal(t, "https://127.0.0.1:8200", after.Location)
}

func TestResolveFailsWithNoStoreClient(t *testing.T) {
	r := &extsecrets.BackendResolver{}

	_, err := r.Resolve(externalSecret("s", "jx", "jx-secret-store"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot determine the secret backend")
}

func TestResolveNilReceiverFails(t *testing.T) {
	var r *extsecrets.BackendResolver

	_, err := r.Resolve(externalSecret("s", "jx", "store"))
	require.Error(t, err)
}
