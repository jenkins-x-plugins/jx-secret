package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ExternalSecret represents a collection of mappings of Secrets to destinations in the underlying secret store (e.g. Vault keys)
//
// +k8s:openapi-gen=true
type ExternalSecret struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata"`

	// Spec holds the desired state of the ExternalSecret from the client
	// +optional
	Spec ExternalSecretSpec `json:"spec"`

	// Status holds the current status
	// +optional
	Status *ExternalSecretStatus `json:"status,omitempty"`
}

// Keys returns the keys for the secret
func (s *ExternalSecret) Keys() []string {
	var keys []string
	if s.Spec.Data != nil {
		for _, d := range s.Spec.Data {
			keys = append(keys, d.Key)
		}
	}
	return keys
}

// KeyAndNames returns the data key and names for the secret
func (s *ExternalSecret) KeyAndNames() []string {
	var keys []string
	if s.Spec.Data != nil {
		for _, d := range s.Spec.Data {
			keys = append(keys, d.Key+"/"+d.Name)
		}
	}
	return keys
}

// ExternalSecretSpec defines the desired state of ExternalSecret.
type ExternalSecretSpec struct {
	BackendType     string `json:"backendType,omitempty"`
	VaultMountPoint string `json:"vaultMountPoint,omitempty"`
	VaultRole       string `json:"vaultRole,omitempty"`
	ProjectID       string `json:"projectId,omitempty"`

	// Data the data for each entry in the Secret
	Data []Data `json:"data,omitempty"`

	// Template
	Template Template `json:"template,omitempty"`
}

// ExternalSecretStatus defines the current status of the ExternalSecret.
type ExternalSecretStatus struct {
	LastSync           metav1.Time `json:"lastSync,omitempty"`
	ObservedGeneration int         `json:"observedGeneration,omitempty"`
	Status             string      `json:"status,omitempty"`
}

// ExternalSecretList contains a list of ExternalSecret
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ExternalSecretList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ExternalSecret `json:"items"`
}

// Data the data properties
type Data struct {
	// Name name of the secret entry
	Name string `json:"name,omitempty"`

	// Key the key in the underlying secret storage (e.g. the key in vault)
	Key string `json:"key,omitempty"`

	// Property the property in the underlying secret storage (e.g.  in vault)
	Property string `json:"property,omitempty"`
}

// Template the template data
type Template struct {
	// Type the type of the secret
	Type string `json:"type,omitempty"`

	// Metadata the metadata such as labels or annotations
	Metadata metav1.ObjectMeta `json:"metadata,omitempty"`
}
