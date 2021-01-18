package editor

import "strings"

// Interface an editor of a secret
type Interface interface {
	Write(properties *KeyProperties) error
}

// KeyProperties to specify a set of properties to populate
type KeyProperties struct {
	Key        string
	Properties []PropertyValue
	// optional GCP project ID where secrets manager is running
	GCPProject string
}

// SecretProperty the property to edit
type PropertyValue struct {
	Property string
	Value    string
	Name     string
}

// String returns a string representation
func (p *KeyProperties) String() string {
	buf := strings.Builder{}
	buf.WriteString("key: ")
	buf.WriteString(p.Key)
	buf.WriteString(" properties: ")
	for i, pv := range p.Properties {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(pv.Property)
	}
	return buf.String()
}
