package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

const (
	LabelKind = "kind"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Schema defines a schema of objects with properties
//
// +k8s:openapi-gen=true
type Schema struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata"`

	// Spec the schema specification
	Spec SchemaSpec `yaml:"spec"`
}

// SchemaList contains a list of Schema objects
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SchemaList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Schema `json:"items"`
}

// SchemaSpec defines the objects and their properties
type SchemaSpec struct {
	// Objects the list of objects (or kinds) in the schema
	Objects []Object `yaml:"objects"`
}

// Object defines a type of object with some properties
type Object struct {
	// Name the name of the object kind
	Name string `yaml:"name" validate:"nonzero"`

	// Properties the property definitions
	Properties []Property `yaml:"properties"`
}

// Property defines a property in an object
type Property struct {
	// Name the name of the property
	Name string `yaml:"name" validate:"nonzero"`

	// Question the main prompt generated in a user interface when asking to populate the property
	Question string `yaml:"question" validate:"nonzero"`

	// Help the tooltip or help text for this property
	Help string `yaml:"help"`

	// DefaultValue is used to specify default values populated on startup
	DefaultValue string `yaml:"defaultValue,omitempty"`

	// Pattern is a regular expression pattern used for validation
	Pattern string `yaml:"pattern,omitempty"`

	// Requires specifies a requirements expression
	Requires string `yaml:"requires,omitempty"`

	// Format the format of the value
	Format string `yaml:"format,omitempty"`

	// Generator the name of the generator to use to create values
	// if this value is non zero we assume Generate is effectively true
	Generator string `yaml:"generator,omitempty"`

	// Labels allows arbitrary metadata labels to be associated with the property
	Labels map[string]string `yaml:"labels,omitempty"`

	// MinLength the minimum number of characters in the value
	MinLength int `yaml:"minLength,omitempty"`

	// MaxLength the maximum number of characters in the value
	MaxLength int `yaml:"maxLength,omitempty"`

	// Mask whether a mask is used on input
	Mask bool `yaml:"mask,omitempty"`
}
