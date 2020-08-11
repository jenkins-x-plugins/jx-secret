package testschemas

import (
	"testing"

	"github.com/jenkins-x/jx-secret/pkg/schema"
	"github.com/stretchr/testify/require"
)

// RequireSchemaProperty finds the mandatory property of the given object schema
func RequireSchemaProperty(t *testing.T, s *schema.Schema, objectName, property string) *schema.Property {
	p, err := schema.FindObjectProperty(s, objectName, property)
	require.NoError(t, err, "failed to find property %s in object %s", property, objectName)
	require.NotNil(t, p, "no property %s found in object %s", property, objectName)
	return p
}
