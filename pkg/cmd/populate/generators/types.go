package generators

import "github.com/jenkins-x/jx-secret/pkg/schema"

// Arguments the generator arguments
type Arguments struct {
	Schema   schema.Schema
	Object   schema.Object
	Property schema.Property
}

// Generator a generator function
type Generator func(Arguments) (string, error)
