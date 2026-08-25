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

// fakeStores serves stores from a map and counts lookups so we can assert the
// resolver caches.
type fakeStores struct {
	stores map[string]esv1.GenericStore
	calls  int
	err    error
}

func (f *fakeStores) GetStore(kind, name, esNamespace string) (esv1.GenericStore, error) {
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

func clusterStore(name string, provider *esv1.SecretStoreProvider) esv1.GenericStore {
	return &esv1.ClusterSecretStore{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec:       esv1.SecretStoreSpec{Provider: provider},
	}
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
			r := &extsecrets.BackendResolver{
				Stores: &fakeStores{stores: map[string]esv1.GenericStore{
					esv1.ClusterSecretStoreKind + "/store": clusterStore("store", tc.provider),
				}},
			}
			es := externalSecret("my-secret", "jx", "store")

			assert.Equal(t, tc.wantBackend, r.Backend(es), "backend")
			assert.Equal(t, tc.wantLocation, r.Location(es), "location")
		})
	}
}

func TestResolveFromStoreBeatsMapping(t *testing.T) {
	r := &extsecrets.BackendResolver{
		Stores: &fakeStores{stores: map[string]esv1.GenericStore{
			esv1.ClusterSecretStoreKind + "/users-own-store": clusterStore("users-own-store",
				&esv1.SecretStoreProvider{GCPSM: &esv1.GCPSMProvider{ProjectID: "users-project"}}),
		}},
		Mapping: &v1alpha1.SecretMapping{
			Spec: v1alpha1.SecretMappingSpec{
				Defaults: v1alpha1.Defaults{BackendType: v1alpha1.BackendTypeVault},
			},
		},
	}

	es := externalSecret("their-secret", "jx", "users-own-store")
	assert.Equal(t, v1alpha1.BackendTypeGSM, r.Backend(es), "a user's own store must win over the jx mapping default")
	assert.Equal(t, "users-project", r.Location(es))
}

func TestResolveFallsBackToMappingWhenStoreUnreadable(t *testing.T) {
	r := &extsecrets.BackendResolver{
		Stores: &fakeStores{err: assert.AnError},
		Mapping: &v1alpha1.SecretMapping{
			Spec: v1alpha1.SecretMappingSpec{
				Defaults: v1alpha1.Defaults{
					BackendType:       v1alpha1.BackendTypeGSM,
					GcpSecretsManager: &v1alpha1.GcpSecretsManager{ProjectID: "jx-project"},
				},
			},
		},
	}

	es := externalSecret("my-secret", "jx", "store")
	assert.Equal(t, v1alpha1.BackendTypeGSM, r.Backend(es))
	assert.Equal(t, "jx-project", r.Location(es))
	assert.Equal(t, "jx-project", r.ProjectID(es))
}

func TestResolveRuleBeatsMappingDefaults(t *testing.T) {
	r := &extsecrets.BackendResolver{
		Mapping: &v1alpha1.SecretMapping{
			Spec: v1alpha1.SecretMappingSpec{
				Secrets: []v1alpha1.SecretRule{
					{
						Name:              "special",
						BackendType:       v1alpha1.BackendTypeGSM,
						GcpSecretsManager: &v1alpha1.GcpSecretsManager{ProjectID: "other-project"},
					},
				},
				Defaults: v1alpha1.Defaults{
					BackendType:       v1alpha1.BackendTypeGSM,
					GcpSecretsManager: &v1alpha1.GcpSecretsManager{ProjectID: "default-project"},
				},
			},
		},
	}

	assert.Equal(t, "other-project", r.Location(externalSecret("special", "jx", "")))
	assert.Equal(t, "default-project", r.Location(externalSecret("ordinary", "jx", "")))
}

// The mapping field convert validates for AWS is secretsManager.region, but
// older mappings use the top-level region; both must resolve.
func TestResolveAWSRegionFromEitherMappingField(t *testing.T) {
	nested := &extsecrets.BackendResolver{
		Mapping: &v1alpha1.SecretMapping{Spec: v1alpha1.SecretMappingSpec{Defaults: v1alpha1.Defaults{
			BackendType:       v1alpha1.BackendTypeAWSSecretsManager,
			AwsSecretsManager: &v1alpha1.AwsSecretsManager{Region: "eu-west-1"},
		}}},
	}
	assert.Equal(t, "eu-west-1", nested.Location(externalSecret("s", "jx", "")))

	topLevel := &extsecrets.BackendResolver{
		Mapping: &v1alpha1.SecretMapping{Spec: v1alpha1.SecretMappingSpec{Defaults: v1alpha1.Defaults{
			BackendType: v1alpha1.BackendTypeAWSSecretsManager,
			Region:      "us-west-2",
		}}},
	}
	assert.Equal(t, "us-west-2", topLevel.Location(externalSecret("s", "jx", "")))
}

// ESO's vault provider takes a mount-relative key, but secretfacade drives the
// Vault HTTP API and needs the full path.
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
			r := &extsecrets.BackendResolver{
				Stores: &fakeStores{stores: map[string]esv1.GenericStore{
					esv1.ClusterSecretStoreKind + "/store": clusterStore("store", tc.provider),
				}},
			}
			assert.Equal(t, tc.want, r.RemoteKeyPath(externalSecret("s", "jx", "store"), tc.key))
		})
	}
}

func TestResolveCachesPerStore(t *testing.T) {
	stores := &fakeStores{stores: map[string]esv1.GenericStore{
		esv1.ClusterSecretStoreKind + "/store": clusterStore("store",
			&esv1.SecretStoreProvider{GCPSM: &esv1.GCPSMProvider{ProjectID: "p"}}),
	}}
	r := &extsecrets.BackendResolver{Stores: stores}

	for i := 0; i < 5; i++ {
		require.Equal(t, v1alpha1.BackendTypeGSM, r.Backend(externalSecret("secret", "jx", "store")))
	}
	assert.Equal(t, 1, stores.calls, "the store should be read once and cached")
}

// A local secret's location is its own namespace, so it must not be cached
// along with the rest of the store's details.
func TestResolveLocalNamespaceIsNotCached(t *testing.T) {
	r := &extsecrets.BackendResolver{
		Mapping: &v1alpha1.SecretMapping{Spec: v1alpha1.SecretMappingSpec{
			Defaults: v1alpha1.Defaults{BackendType: v1alpha1.BackendTypeLocal},
		}},
	}

	assert.Equal(t, "jx", r.Location(externalSecret("s", "jx", "")))
	assert.Equal(t, "jx-staging", r.Location(externalSecret("s", "jx-staging", "")))
}

// VAULT_ADDR is set partway through a populate run by the vault port-forward,
// so the location must be read on access rather than frozen at resolve time.
func TestResolveVaultAddressIsReadOnAccess(t *testing.T) {
	r := &extsecrets.BackendResolver{
		Mapping: &v1alpha1.SecretMapping{Spec: v1alpha1.SecretMappingSpec{
			Defaults: v1alpha1.Defaults{BackendType: v1alpha1.BackendTypeVault},
		}},
	}
	es := externalSecret("s", "jx", "")

	require.Empty(t, r.Location(es))
	t.Setenv("VAULT_ADDR", "https://127.0.0.1:8200")
	assert.Equal(t, "https://127.0.0.1:8200", r.Location(es))
}

func TestResolveNilSafe(t *testing.T) {
	var r *extsecrets.BackendResolver
	es := externalSecret("s", "jx", "store")

	assert.Equal(t, v1alpha1.BackendTypeNone, r.Backend(es))
	assert.Empty(t, r.Location(es))
	assert.Empty(t, r.ProjectID(es))
	assert.Equal(t, "some/key", r.RemoteKeyPath(es, "some/key"))
}

// A provider jx-secret cannot write to must not be mistaken for the jx backend.
func TestResolveUnsupportedProviderIsEmpty(t *testing.T) {
	r := &extsecrets.BackendResolver{
		Stores: &fakeStores{stores: map[string]esv1.GenericStore{
			esv1.ClusterSecretStoreKind + "/store": clusterStore("store",
				&esv1.SecretStoreProvider{Fake: &esv1.FakeProvider{}}),
		}},
	}
	assert.Equal(t, v1alpha1.BackendTypeNone, r.Backend(externalSecret("s", "jx", "store")))
}
