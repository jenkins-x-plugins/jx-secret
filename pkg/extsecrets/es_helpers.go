package extsecrets

import (
	"fmt"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

// DefaultSecretStoreName is the ClusterSecretStore the jx3-versions chart ships
// in every cluster, and the store every emitted ExternalSecret references.
// Hardcoding the name keeps jx-secret decoupled from the BackendType enum,
// since the store carries the backend-specific config.
const DefaultSecretStoreName = "jx-secret-store"

// DefaultSecretStoreKind is the kind of the referenced secret store.
const DefaultSecretStoreKind = "ClusterSecretStore"

// APIVersion is stamped onto every emitted ExternalSecret.
var APIVersion = esv1.SchemeGroupVersion.String()

// KeyAndNames returns "<remoteRef.key>/<secretKey>" for each data entry.
func KeyAndNames(es *esv1.ExternalSecret) []string {
	var keys []string
	for i := range es.Spec.Data {
		d := &es.Spec.Data[i]
		keys = append(keys, d.RemoteRef.Key+"/"+d.SecretKey)
	}
	return keys
}

// KeyAndProperty returns the remote-ref key and property of the data entry
// producing the given secret key.
func KeyAndProperty(es *esv1.ExternalSecret, secretKey string) (string, string, error) {
	for i := range es.Spec.Data {
		d := &es.Spec.Data[i]
		if d.SecretKey == secretKey {
			return d.RemoteRef.Key, d.RemoteRef.Property, nil
		}
	}
	return "", "", fmt.Errorf("unable to find secret data entry of %s of External Secret %s", secretKey, es.Name)
}

// SecretLocation returns a stable identity string for a data entry. The backend
// is a parameter because the ExternalSecret does not carry it; resolve it with
// BackendResolver.Resolve.
func SecretLocation(backend string, d *esv1.ExternalSecretData) string {
	return fmt.Sprintf("%s/%s/%s/%s", backend, d.RemoteRef.Key, d.RemoteRef.Property, d.RemoteRef.Version)
}
