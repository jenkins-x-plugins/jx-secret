package generators

import (
	"github.com/jenkins-x/jx-secret/pkg/apis/schema/v1alpha1"
)

// Arguments the generator arguments
type Arguments struct {
	Schema   *v1alpha1.Schema
	Object   *v1alpha1.Object
	Property *v1alpha1.Property
}

// Generator a generator function
type Generator func(*Arguments) (string, error)
