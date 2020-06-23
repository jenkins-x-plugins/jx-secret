package secretfacade

import (
	"github.com/jenkins-x/jx-extsecret/pkg/apis/extsecret/v1alpha1"
	"github.com/jenkins-x/jx-extsecret/pkg/extsecrets"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// Options options for verifying secrets
type Options struct {
	SecretClient extsecrets.Interface
	KubeClient   kubernetes.Interface
	Namespace    string

	// ExternalSecrets the loaded secrets
	ExternalSecrets []*v1alpha1.ExternalSecret
}

// SecretError returns an error for a secret
type SecretError struct {
	// ExternalSecret the external secret which is not valid
	ExternalSecret v1alpha1.ExternalSecret

	// EntryErrors the errors for each secret entry
	EntryErrors []*EntryError
}

// EntryError represents the missing entries
type EntryError struct {
	// Key the secret key
	Key string

	// Properties property names for the key
	Properties []string
}

// SecretPair the external secret and the associated Secret an error for a secret
type SecretPair struct {
	// ExternalSecret the external secret which is not valid
	ExternalSecret v1alpha1.ExternalSecret

	// Secret the secret if there is one
	Secret *corev1.Secret
}
