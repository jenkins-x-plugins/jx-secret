package extsecrets

import (
	"fmt"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

// DefaultSecretStoreName is the ClusterSecretStore name that the jx3-versions
// ESO chart ships in every cluster. jx-secret hardcodes this name so it stays
// decoupled from the SecretMapping BackendType enum — the store itself carries
// backend-specific config.
const DefaultSecretStoreName = "jx-secret-store"

// Keys returns the remote-ref keys for each data entry of the given
// ExternalSecret.
func Keys(es *esv1.ExternalSecret) []string {
	var keys []string
	for _, d := range es.Spec.Data {
		keys = append(keys, d.RemoteRef.Key)
	}
	return keys
}

// KeyAndNames returns "<remoteRef.key>/<secretKey>" for each data entry.
func KeyAndNames(es *esv1.ExternalSecret) []string {
	var keys []string
	for _, d := range es.Spec.Data {
		keys = append(keys, d.RemoteRef.Key+"/"+d.SecretKey)
	}
	return keys
}

// KeyAndProperty looks up a data entry by its produced secret key and returns
// (remoteRef.key, remoteRef.property).
func KeyAndProperty(es *esv1.ExternalSecret, secretKey string) (string, string, error) {
	for _, d := range es.Spec.Data {
		if d.SecretKey == secretKey {
			return d.RemoteRef.Key, d.RemoteRef.Property, nil
		}
	}
	return "", "", fmt.Errorf("unable to find secret data entry of %s of External Secret %s", secretKey, es.Name)
}

// SecretLocation returns a stable identity string for a data entry. Backend is
// passed in because the ExternalSecret no longer carries backend info directly —
// callers resolve it via ResolveBackend (see resolve_backend.go).
func SecretLocation(backend string, d esv1.ExternalSecretData) string {
	return fmt.Sprintf("%s/%s/%s/%s", backend, d.RemoteRef.Key, d.RemoteRef.Property, d.RemoteRef.Version)
}