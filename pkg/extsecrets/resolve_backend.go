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

	// GCP project, Azure vault name, AWS region, or namespace for local secrets
	location string

	VaultMount string
	VaultKVv2  bool
}

// Location is what secretfacade takes as its location. Vault reads VAULT_ADDR on each call,
// as the store's server is ESO's in-cluster address and populate sets VAULT_ADDR after resolving.
func (b *Backend) Location() string {
	if b.Type == v1alpha1.BackendTypeVault {
		return os.Getenv("VAULT_ADDR")
	}
	return b.location
}

// RemoteKeyPath converts a remoteRef.key into the full path secretfacade needs.
// Mirrors buildPath in ESO's vault provider, or the two silently address different secrets.
func (b *Backend) RemoteKeyPath(key string) string {
	if b == nil || b.Type != v1alpha1.BackendTypeVault || key == "" {
		return key
	}
	if b.VaultMount == "" {
		// with no mount on the store, ESO treats the key's first segment as the mount
		if !b.VaultKVv2 || strings.Contains(key, "/data/") {
			return key
		}
		segments := strings.Split(key, "/")
		return strings.Join(append([]string{segments[0], "data"}, segments[1:]...), "/")
	}

	rest, found := strings.CutPrefix(key, b.VaultMount+"/")
	if found && b.VaultKVv2 {
		rest = strings.TrimPrefix(rest, "data/")
	}
	if b.VaultKVv2 {
		return b.VaultMount + "/data/" + rest
	}
	return b.VaultMount + "/" + rest
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

// fillDynamic sets a local secret's namespace, which varies per ExternalSecret so cannot be cached.
func (r *BackendResolver) fillDynamic(b *Backend, es *esv1.ExternalSecret) *Backend {
	if b.Type != v1alpha1.BackendTypeLocal {
		return b
	}
	clone := *b
	clone.location = es.Namespace
	return &clone
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
			Type: v1alpha1.BackendTypeVault,
			// ESO treats an unset version as v2
			VaultKVv2: p.Vault.Version != esv1.VaultKVStoreV1,
		}
		if p.Vault.Path != nil && *p.Vault.Path != "" {
			b.VaultMount = strings.Trim(*p.Vault.Path, "/")
		}
		return b

	case p.GCPSM != nil:
		return &Backend{Type: v1alpha1.BackendTypeGSM, location: p.GCPSM.ProjectID}

	case p.AzureKV != nil:
		return &Backend{Type: v1alpha1.BackendTypeAzure, location: azureVaultName(p.AzureKV.VaultURL)}

	case p.AWS != nil:
		backendType := v1alpha1.BackendTypeAWSSecretsManager
		if p.AWS.Service == esv1.AWSServiceParameterStore {
			backendType = v1alpha1.BackendTypeAWSParameterStore
		}
		return &Backend{Type: backendType, location: p.AWS.Region}

	case p.Kubernetes != nil:
		// remoteNamespace is deliberately ignored: it is where ESO would read from,
		// but jx-secret writes the target Secret itself, next to the ExternalSecret.
		// The CRD defaults it to "default", so honouring it would misplace every
		// local secret.
		return &Backend{Type: v1alpha1.BackendTypeLocal}

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
