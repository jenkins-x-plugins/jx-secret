package extsecrets

import (
	"net/url"
	"os"
	"strings"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/jenkins-x-plugins/jx-secret/pkg/apis/mapping/v1alpha1"
	"github.com/pkg/errors"
)

// A BackendResolver derives, for a given ExternalSecret, the information
// jx-secret needs in order to talk to the secret backend directly: which
// backend it is, where it lives, and the backend-native path for a remote ref.
type BackendResolver struct {
	// Stores reads the (Cluster)SecretStore referenced by an ExternalSecret.
	Stores StoreInterface

	// storeCache memoises successful store lookups per ref so a run over many
	// ExternalSecrets makes one API call per distinct store. Failures are not
	// cached because they end the run.
	storeCache map[string]*Backend
}

// A Backend describes a resolved secret backend.
type Backend struct {
	// Type is the jx backend type, mapped from the store's provider.
	Type v1alpha1.BackendType

	// Location is what the secretfacade store manager takes as its location
	// argument: GCP project ID, Azure key-vault name, Vault server address, AWS
	// region, or namespace for local secrets.
	Location string

	// VaultMount is the Vault KV mount path, e.g. "secret". Empty for other
	// backends.
	VaultMount string

	// VaultKVv2 reports whether the Vault mount is KV version 2, which needs
	// the "data" path segment injected when reading and writing via the API.
	VaultKVv2 bool
}

// DefaultVaultMount is the KV mount assumed when a store does not name one.
const DefaultVaultMount = "secret"

// RemoteKeyPath converts an ExternalSecret remoteRef.key into the path the
// backend's API expects.
//
// Only Vault differs: ESO's vault provider takes a mount-relative path and adds
// the KV v2 "data" segment itself, but secretfacade drives the Vault HTTP API
// directly and needs the full path. So "jx/pipelineUser" against a KV v2 mount
// named "secret" becomes "secret/data/jx/pipelineUser".
func (b *Backend) RemoteKeyPath(key string) string {
	if b == nil || b.Type != v1alpha1.BackendTypeVault || key == "" {
		return key
	}
	mount := b.VaultMount
	if mount == "" {
		mount = DefaultVaultMount
	}
	// tolerate keys that already carry the mount, e.g. mappings written by hand
	// against the pre-ESO layout
	if strings.HasPrefix(key, mount+"/") {
		return key
	}
	if b.VaultKVv2 {
		return mount + "/data/" + key
	}
	return mount + "/" + key
}

// Resolve returns the backend for the given ExternalSecret, or an error if it
// cannot be determined. Resolve once per ExternalSecret and pass the result
// down rather than resolving per key.
func (r *BackendResolver) Resolve(es *esv1.ExternalSecret) (*Backend, error) {
	if r == nil {
		return nil, errors.New("no backend resolver configured")
	}
	if es == nil {
		return nil, errors.New("no ExternalSecret given")
	}

	if r.Stores == nil {
		return nil, errors.Errorf("cannot determine the secret backend for ExternalSecret %s: no SecretStore client configured", esID(es))
	}
	ref := es.Spec.SecretStoreRef
	if ref.Name == "" {
		return nil, errors.Errorf("cannot determine the secret backend for ExternalSecret %s: it references no SecretStore", esID(es))
	}

	b, err := r.fromStore(es, ref)
	if err != nil {
		return nil, err
	}
	return r.fillDynamic(b, es), nil
}

func (r *BackendResolver) fromStore(es *esv1.ExternalSecret, ref esv1.SecretStoreRef) (*Backend, error) {
	kind := ref.Kind
	if kind == "" {
		// ESO defaults an unset kind to the namespaced SecretStore
		kind = esv1.SecretStoreKind
	}
	cacheKey := kind + "/" + ref.Name + "/" + es.Namespace
	if b, ok := r.storeCache[cacheKey]; ok {
		return b, nil
	}

	store, err := r.Stores.GetStore(ref.Kind, ref.Name, es.Namespace)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read the %s %s referenced by ExternalSecret %s, which holds the secret backend configuration. Check the store exists and that jx-secret is allowed to get %ss", kind, ref.Name, esID(es), strings.ToLower(kind))
	}
	b := backendFromStore(store)
	if b.Type == "" {
		return nil, errors.Errorf("the %s %s referenced by ExternalSecret %s uses a provider jx-secret cannot write to", kind, ref.Name, esID(es))
	}

	if r.storeCache == nil {
		r.storeCache = map[string]*Backend{}
	}
	r.storeCache[cacheKey] = b
	return b, nil
}

// fillDynamic supplies the parts of a Backend that must not be cached: the
// namespace of a local secret varies per ExternalSecret, and VAULT_ADDR is set
// partway through a populate run by the vault port-forward, so reading it at
// resolve time would freeze in whatever was set beforehand.
func (r *BackendResolver) fillDynamic(b *Backend, es *esv1.ExternalSecret) *Backend {
	if b.Location != "" {
		return b
	}
	switch b.Type {
	case v1alpha1.BackendTypeLocal:
		clone := *b
		clone.Location = es.Namespace
		return &clone
	case v1alpha1.BackendTypeVault:
		clone := *b
		clone.Location = os.Getenv("VAULT_ADDR")
		return &clone
	}
	return b
}

func esID(es *esv1.ExternalSecret) string {
	if es.Namespace == "" {
		return es.Name
	}
	return es.Namespace + "/" + es.Name
}

// backendFromStore maps an ESO store provider onto a jx backend. Only the
// providers jx-secret can write to via secretfacade are mapped; anything else
// resolves empty so the caller reports it rather than writing to the wrong
// place.
func backendFromStore(store esv1.GenericStore) *Backend {
	spec := store.GetSpec()
	if spec == nil || spec.Provider == nil {
		return &Backend{}
	}
	p := spec.Provider

	switch {
	case p.Vault != nil:
		b := &Backend{
			Type:       v1alpha1.BackendTypeVault,
			Location:   p.Vault.Server,
			VaultMount: DefaultVaultMount,
			// ESO treats an unset version as v2, and so do we
			VaultKVv2: p.Vault.Version != esv1.VaultKVStoreV1,
		}
		if p.Vault.Path != nil && *p.Vault.Path != "" {
			b.VaultMount = strings.Trim(*p.Vault.Path, "/")
		}
		// an empty server is filled from VAULT_ADDR by fillDynamic
		return b

	case p.GCPSM != nil:
		return &Backend{Type: v1alpha1.BackendTypeGSM, Location: p.GCPSM.ProjectID}

	case p.AzureKV != nil:
		return &Backend{Type: v1alpha1.BackendTypeAzure, Location: azureVaultName(p.AzureKV.VaultURL)}

	case p.AWS != nil:
		backendType := v1alpha1.BackendTypeAWSSecretsManager
		if p.AWS.Service == esv1.AWSServiceParameterStore {
			backendType = v1alpha1.BackendTypeAWSParameterStore
		}
		return &Backend{Type: backendType, Location: p.AWS.Region}

	case p.Kubernetes != nil:
		return &Backend{Type: v1alpha1.BackendTypeLocal, Location: p.Kubernetes.RemoteNamespace}

	case p.IBM != nil:
		return &Backend{Type: v1alpha1.BackendTypeIBMSecretsManager}
	}
	return &Backend{}
}

// azureVaultName reduces a key-vault URL to the bare vault name, which is what
// secretfacade's Azure store manager takes as its location.
func azureVaultName(vaultURL *string) string {
	if vaultURL == nil || *vaultURL == "" {
		return ""
	}
	u, err := url.Parse(*vaultURL)
	if err != nil || u.Hostname() == "" {
		return ""
	}
	return strings.SplitN(u.Hostname(), ".", 2)[0]
}
