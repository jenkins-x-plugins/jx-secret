package extsecrets

import (
	"net/url"
	"os"
	"strings"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/jenkins-x-plugins/jx-secret/pkg/apis/mapping/v1alpha1"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
)

// A BackendResolver derives, for a given ExternalSecret, the information
// jx-secret needs in order to talk to the secret backend directly: which
// backend it is, where it lives, and the backend-native path for a remote ref.
//
// The ExternalSecret carries none of this itself — it lives on the
// (Cluster)SecretStore named by spec.secretStoreRef — so Stores is the
// authoritative source. That matters beyond correctness: users bring their own
// stores for their own secrets, and reading the store is the only way to honour
// them rather than assuming everything lives in the jx backend.
//
// Mapping is a fallback for when no store can be read: the `convert` command
// runs before any store exists, `--source filesystem` has no cluster at all,
// and tests supply a mapping directly. It holds the same information for
// jx-managed secrets, so resolution degrades rather than failing outright.
type BackendResolver struct {
	// Stores reads the (Cluster)SecretStore referenced by an ExternalSecret.
	// When nil, resolution falls back to Mapping.
	Stores StoreInterface

	// Mapping is consulted when Stores is nil or the referenced store cannot be
	// read.
	Mapping *v1alpha1.SecretMapping

	// storeCache memoises the store lookup per ref so a run over many
	// ExternalSecrets makes one API call per distinct store. A present entry
	// holding nil means the store was looked up and yielded nothing usable.
	// Only store results are cached: mapping resolution varies per
	// ExternalSecret name and is local anyway.
	storeCache map[string]*Backend

	// warned tracks store refs we have already reported as unreadable, to keep
	// the fallback from logging once per ExternalSecret.
	warned map[string]bool
}

// A Backend describes a resolved secret backend. The zero value means
// unresolved, which callers should treat as "cannot reach this backend".
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

// IsEmpty reports whether nothing could be resolved.
func (b *Backend) IsEmpty() bool {
	return b == nil || b.Type == ""
}

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

// DefaultVaultMount is the KV mount assumed when a store does not name one.
const DefaultVaultMount = "secret"

// Resolve returns the backend for the given ExternalSecret. It never returns
// nil, so callers can chain field access safely.
func (r *BackendResolver) Resolve(es *esv1.ExternalSecret) *Backend {
	if r == nil || es == nil {
		return &Backend{}
	}

	if b := r.backendFromStoreRef(es); b != nil {
		return r.fillDynamic(b, es)
	}
	return r.fillDynamic(r.backendFromMapping(es), es)
}

// backendFromStoreRef resolves via the referenced store, returning nil when
// there is no readable store to resolve from so the caller falls back to the
// mapping.
func (r *BackendResolver) backendFromStoreRef(es *esv1.ExternalSecret) *Backend {
	ref := es.Spec.SecretStoreRef
	if r.Stores == nil || ref.Name == "" {
		return nil
	}

	cacheKey := ref.Kind + "/" + ref.Name + "/" + es.Namespace
	if b, ok := r.storeCache[cacheKey]; ok {
		return b
	}

	var resolved *Backend
	store, err := r.Stores.GetStore(ref.Kind, ref.Name, es.Namespace)
	switch {
	case err != nil:
		r.warnOncef(cacheKey, "could not read %s %s (%s); falling back to the SecretMapping", ref.Kind, ref.Name, err.Error())
	default:
		if b := backendFromStore(store); !b.IsEmpty() {
			resolved = b
		} else {
			r.warnOncef(cacheKey, "store %s %s has no provider jx-secret can write to; falling back to the SecretMapping", ref.Kind, ref.Name)
		}
	}

	if r.storeCache == nil {
		r.storeCache = map[string]*Backend{}
	}
	r.storeCache[cacheKey] = resolved
	return resolved
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

func (r *BackendResolver) warnOncef(key, format string, args ...interface{}) {
	if r.warned == nil {
		r.warned = map[string]bool{}
	}
	if r.warned[key] {
		return
	}
	r.warned[key] = true
	log.Logger().Warnf(format, args...)
}

// backendFromStore maps an ESO store provider onto a jx backend. Only the
// providers jx-secret can write to via secretfacade are mapped; anything else
// resolves empty so the caller can skip rather than write to the wrong place.
func backendFromStore(store esv1.GenericStore) *Backend {
	spec := store.GetSpec()
	if spec == nil {
		return &Backend{}
	}
	p := spec.Provider
	if p == nil {
		return &Backend{}
	}

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
		backend := v1alpha1.BackendTypeAWSSecretsManager
		if p.AWS.Service == esv1.AWSServiceParameterStore {
			backend = v1alpha1.BackendTypeAWSParameterStore
		}
		return &Backend{Type: backend, Location: p.AWS.Region}

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
	raw := *vaultURL
	if !strings.Contains(raw, "://") {
		// already a bare name, or a host without a scheme
		return strings.SplitN(strings.TrimSuffix(raw, "/"), ".", 2)[0]
	}
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" {
		return ""
	}
	return strings.SplitN(u.Hostname(), ".", 2)[0]
}

// backendFromMapping resolves from the SecretMapping. A rule-level value takes
// precedence over the mapping-wide default.
func (r *BackendResolver) backendFromMapping(es *esv1.ExternalSecret) *Backend {
	if r.Mapping == nil {
		return &Backend{}
	}
	rule := r.Mapping.FindRule(es.Namespace, es.Name)
	defaults := r.Mapping.Spec.Defaults

	backend := defaults.BackendType
	if rule != nil && rule.BackendType != "" {
		backend = rule.BackendType
	}

	b := &Backend{Type: backend}
	switch backend {
	case v1alpha1.BackendTypeGSM:
		if rule != nil && rule.GcpSecretsManager != nil && rule.GcpSecretsManager.ProjectID != "" {
			b.Location = rule.GcpSecretsManager.ProjectID
		} else if defaults.GcpSecretsManager != nil {
			b.Location = defaults.GcpSecretsManager.ProjectID
		}
	case v1alpha1.BackendTypeAzure:
		if rule != nil && rule.AzureKeyVaultConfig != nil && rule.AzureKeyVaultConfig.KeyVaultName != "" {
			b.Location = rule.AzureKeyVaultConfig.KeyVaultName
		} else if defaults.AzureKeyVaultConfig != nil {
			b.Location = defaults.AzureKeyVaultConfig.KeyVaultName
		}
	case v1alpha1.BackendTypeVault:
		// location comes from VAULT_ADDR via fillDynamic
		b.VaultMount = DefaultVaultMount
		b.VaultKVv2 = true
	case v1alpha1.BackendTypeAWSSecretsManager, v1alpha1.BackendTypeAWSParameterStore:
		b.Location = awsRegionFromMapping(rule, &defaults)
	case v1alpha1.BackendTypeLocal:
		b.Location = es.Namespace
	}
	return b
}

// awsRegionFromMapping checks both the `secretsManager.region` field that
// `convert` validates and the older top-level `region`, most specific first.
func awsRegionFromMapping(rule *v1alpha1.SecretRule, defaults *v1alpha1.Defaults) string {
	if rule != nil {
		if rule.AwsSecretsManager != nil && rule.AwsSecretsManager.Region != "" {
			return rule.AwsSecretsManager.Region
		}
		if rule.Region != "" {
			return rule.Region
		}
	}
	if defaults.AwsSecretsManager != nil && defaults.AwsSecretsManager.Region != "" {
		return defaults.AwsSecretsManager.Region
	}
	return defaults.Region
}

// Backend returns the backend type for the given ExternalSecret.
func (r *BackendResolver) Backend(es *esv1.ExternalSecret) v1alpha1.BackendType {
	return r.Resolve(es).Type
}

// Location returns where the secrets live for the given ExternalSecret.
func (r *BackendResolver) Location(es *esv1.ExternalSecret) string {
	return r.Resolve(es).Location
}

// ProjectID returns the GCP project ID for the given ExternalSecret, or empty
// if the resolved backend is not GSM.
func (r *BackendResolver) ProjectID(es *esv1.ExternalSecret) string {
	b := r.Resolve(es)
	if b.Type != v1alpha1.BackendTypeGSM {
		return ""
	}
	return b.Location
}

// RemoteKeyPath returns the backend-native path for one of the
// ExternalSecret's remote-ref keys.
func (r *BackendResolver) RemoteKeyPath(es *esv1.ExternalSecret, key string) string {
	return r.Resolve(es).RemoteKeyPath(key)
}
