package extsecrets

import (
	"net/url"
	"os"
	"strings"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/jenkins-x-plugins/jx-secret/pkg/apis/mapping/v1alpha1"
	"github.com/pkg/errors"
)

// A BackendResolver maps an ExternalSecret onto the backend jx-secret writes to,
// which ESO keeps on the referenced (Cluster)SecretStore rather than the secret.
type BackendResolver struct {
	Stores StoreInterface

	// one lookup per distinct store per run; failures are not cached as they end the run
	storeCache map[string]*Backend
}

// A Backend describes a resolved secret backend.
type Backend struct {
	Type v1alpha1.BackendType

	// what secretfacade takes as its location: GCP project, Azure vault name,
	// Vault server address, AWS region, or namespace for local secrets
	Location string

	VaultMount string
	VaultKVv2  bool
}

// DefaultVaultMount is the KV mount assumed when a store does not name one.
const DefaultVaultMount = "secret"

// RemoteKeyPath converts a remoteRef.key into the path the backend's API expects.
// Only Vault differs: ESO's provider takes a mount-relative path and injects the
// KV v2 "data" segment itself, but secretfacade drives the Vault HTTP API and
// needs the full path.
func (b *Backend) RemoteKeyPath(key string) string {
	if b == nil || b.Type != v1alpha1.BackendTypeVault || key == "" {
		return key
	}
	mount := b.VaultMount
	if mount == "" {
		mount = DefaultVaultMount
	}
	// tolerate hand-written mappings that already spell out the mount
	if strings.HasPrefix(key, mount+"/") {
		return key
	}
	if b.VaultKVv2 {
		return mount + "/data/" + key
	}
	return mount + "/" + key
}

// Resolve returns the backend for the given ExternalSecret. Call it once per
// ExternalSecret and pass the result down rather than resolving per key.
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

// fillDynamic supplies the parts of a Backend that must not be cached: a local
// secret's namespace varies per ExternalSecret, and VAULT_ADDR is only set once
// the vault port-forward comes up, partway through a populate run.
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

// backendFromStore maps an ESO provider onto a jx backend. Only providers
// secretfacade can write to are mapped; anything else resolves empty so the
// caller reports it rather than writing to the wrong place.
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
			// ESO treats an unset version as v2
			VaultKVv2: p.Vault.Version != esv1.VaultKVStoreV1,
		}
		if p.Vault.Path != nil && *p.Vault.Path != "" {
			b.VaultMount = strings.Trim(*p.Vault.Path, "/")
		}
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

// azureVaultName reduces a key-vault URL to the bare name secretfacade's Azure
// store manager takes as its location.
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
