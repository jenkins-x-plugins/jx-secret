package schema

import (
	"gopkg.in/validator.v2"
)

const (
	LabelKind = "kind"
)

// Schema defines a schema of objects with properties
type Schema struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Spec       Spec   `yaml:"spec"`
}

// SchemaSpec defines the objects and their properties
type Spec struct {
	Objects []Object `yaml:"objects"`
}

// Object defines a type of object with some properties
type Object struct {
	Name       string     `yaml:"name" validate:"nonzero"`
	Properties []Property `yaml:"properties"`
}

// Property defines a property in an object
type Property struct {
	Name         string `yaml:"name" validate:"nonzero"`
	Question     string `yaml:"question" validate:"nonzero"`
	Help         string `yaml:"help"`
	DefaultValue string `yaml:"defaultValue,omitempty"`
	Pattern      string `yaml:"pattern,omitempty"`
	Requires     string `yaml:"requires,omitempty"`
	Format       string `yaml:"format,omitempty"`

	// Generator the name of the generator to use to create values
	// if this value is non zero we assume Generate is effectively true
	Generator string `yaml:"generator,omitempty"`

	Labels    map[string]string `yaml:"labels,omitempty"`
	MinLength int               `yaml:"minLength,omitempty"`
	MaxLength int               `yaml:"maxLength,omitempty"`
	Mask      bool              `yaml:"mask,omitempty"`

	// Generate if enabled the secret value will be automatically generated based on the format
	// and other properties
	Generate bool `yaml:"generate,omitempty"`
}

// validate the survey schema fields
func (c *Schema) Validate() error {
	return validator.Validate(c)
}
