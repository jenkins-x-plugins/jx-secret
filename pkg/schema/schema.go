package schema

import (
	"gopkg.in/validator.v2"
)

const (
	LabelSecretKey      = "secretKey"
	LabelSecretProperty = "secretProperty"
	LabelKind           = "kind"
)

type SurveySchema struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Spec       Spec   `json:"spec"`
}

type Survey struct {
	Name         string            `json:"name" validate:"nonzero"`
	Labels       map[string]string `json:"labels,omitempty"`
	Question     string            `json:"question" validate:"nonzero"`
	Help         string            `json:"help"`
	DefaultValue string            `json:"defaultValue,omitempty"`
	Mask         bool              `json:"mask,omitempty"`
	MinLength    int               `json:"minLength,omitempty"`
	MaxLength    int               `json:"maxLength,omitempty"`
	Pattern      string            `json:"pattern,omitempty"`
	Requires     string            `json:"requires,omitempty"`
	Format       string            `json:"format,omitempty"`
}

type Spec struct {
	Survey []Survey `json:"survey"`
}

// validate the survey schema fields
func (c *SurveySchema) Validate() error {
	return validator.Validate(c)
}
