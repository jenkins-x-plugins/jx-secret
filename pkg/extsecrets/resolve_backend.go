package extsecrets

import (
	"os"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/jenkins-x-plugins/jx-secret/pkg/apis/mapping/v1alpha1"
)

// BackendResolver derives backend info for an ESO ExternalSecret from the
// SecretMapping input config that jx-secret loads from `--dir`. After the KES →
// ESO migration the ExternalSecret no longer carries backend info directly —
// it lives on the referenced (Cluster)SecretStore — but jx-secret's own
// SecretMapping already carries the same information (that's what convert.go
// stamps onto the ClusterSecretStore in jx3-versions). So we resolve backend
// from SecretMapping at CLI-run time.
//
// Rule-level override takes precedence over the mapping-wide default
// (SecretMapping.Spec.Defaults, embedded as SecretMapping.Spec.*).
//
// The resolver returns "" from every method when the mapping is nil so
// callers can safely default to no-op behavior when jx-secret is invoked
// outside a mapped dir. Callers who need to fail loudly on missing mapping
// should check that at Options.Validate() time.
type BackendResolver struct {
	Mapping *v1alpha1.SecretMapping
}

// Backend returns the backend type for the given ExternalSecret.
func (r *BackendResolver) Backend(es *esv1.ExternalSecret) v1alpha1.BackendType {
	if r == nil || r.Mapping == nil {
		return ""
	}
	rule := r.Mapping.FindRule(es.Namespace, es.Name)
	if rule != nil && rule.BackendType != "" {
		return rule.BackendType
	}
	return r.Mapping.Spec.BackendType
}

// Location returns the backend-specific "location" string used to identify
// where secrets live for a given ExternalSecret — GCP project ID for GSM,
// key-vault name for Azure, VAULT_ADDR for Vault, AWS region for
// SecretsManager, namespace for local. Mirrors the pre-migration
// populate.GetExternalSecretLocation.
func (r *BackendResolver) Location(es *esv1.ExternalSecret) string {
	if r == nil || r.Mapping == nil {
		return ""
	}
	rule := r.Mapping.FindRule(es.Namespace, es.Name)
	defaults := r.Mapping.Spec.Defaults

	switch r.Backend(es) {
	case v1alpha1.BackendTypeGSM:
		if rule != nil && rule.GcpSecretsManager != nil && rule.GcpSecretsManager.ProjectID != "" {
			return rule.GcpSecretsManager.ProjectID
		}
		if defaults.GcpSecretsManager != nil {
			return defaults.GcpSecretsManager.ProjectID
		}
		return ""
	case v1alpha1.BackendTypeAzure:
		if rule != nil && rule.AzureKeyVaultConfig != nil && rule.AzureKeyVaultConfig.KeyVaultName != "" {
			return rule.AzureKeyVaultConfig.KeyVaultName
		}
		if defaults.AzureKeyVaultConfig != nil {
			return defaults.AzureKeyVaultConfig.KeyVaultName
		}
		return ""
	case v1alpha1.BackendTypeVault:
		return os.Getenv("VAULT_ADDR")
	case v1alpha1.BackendTypeAWSSecretsManager:
		if rule != nil && rule.Region != "" {
			return rule.Region
		}
		return defaults.Region
	case v1alpha1.BackendTypeLocal:
		return es.Namespace
	}
	return ""
}

// ProjectID returns the GCP project ID for the given ExternalSecret if the
// resolved backend is GSM, otherwise empty. Callers previously read
// ExternalSecret.Spec.ProjectID directly.
func (r *BackendResolver) ProjectID(es *esv1.ExternalSecret) string {
	if r.Backend(es) != v1alpha1.BackendTypeGSM {
		return ""
	}
	return r.Location(es)
}

// ResolveBackend is a thin wrapper preserved so pre-refactor callsites still
// compile while the sweep is in progress. Callers with access to
// secretfacade.Options should prefer o.Resolver.Backend(es) — this free
// function is a bridge, not the target API.
//
// TODO(eso-migration): remove once the populate/edit sweep has migrated all
// callers to use o.Resolver directly.
func ResolveBackend(es *esv1.ExternalSecret) v1alpha1.BackendType {
	_ = es
	return ""
}
