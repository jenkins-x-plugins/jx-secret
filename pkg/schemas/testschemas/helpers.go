package testschemas

import (
	"testing"

	"github.com/jenkins-x-plugins/jx-secret/pkg/apis/schema/v1alpha1"
	"github.com/jenkins-x-plugins/jx-secret/pkg/schemas"
	"github.com/stretchr/testify/require"
)

// RequireSchemaProperty finds the mandatory property of the given object schema
func RequireSchemaProperty(t *testing.T, s *v1alpha1.Schema, objectName, property string) *v1alpha1.Property {
	_, p := schemas.FindObjectProperty(s, objectName, property)
	require.NotNil(t, p, "no property %s found in object %s", property, objectName)
	return p
}
