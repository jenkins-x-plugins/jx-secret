package extsecrets

import (
	"os"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/jenkins-x-plugins/jx-secret/pkg/apis/mapping/v1alpha1"
)

// A BackendResolver derives backend info for an ExternalSecret from the
// SecretMapping loaded from `--dir`. The ExternalSecret itself carries none —
// that lives on the referenced (Cluster)SecretStore — but the SecretMapping
// holds the same information, so we resolve from there at CLI-run time.
//
// A rule-level value takes precedence over the mapping-wide default. Every
// method returns "" for a nil mapping so callers invoked outside a mapped dir
// degrade to no-op rather than panic; check for a missing mapping in
// Options.Validate() if you need to fail loudly.
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

// Location returns where the secrets live for the given ExternalSecret: GCP
// project ID for GSM, key-vault name for Azure, VAULT_ADDR for Vault, region
// for AWS SecretsManager, namespace for local.
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

// ProjectID returns the GCP project ID for the given ExternalSecret, or empty
// if the resolved backend is not GSM.
func (r *BackendResolver) ProjectID(es *esv1.ExternalSecret) string {
	if r == nil || r.Mapping == nil {
		return ""
	}
	if r.Backend(es) != v1alpha1.BackendTypeGSM {
		return ""
	}
	return r.Location(es)
}
