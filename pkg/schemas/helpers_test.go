package schemas_test

import (
	"path/filepath"
	"testing"

	"github.com/jenkins-x-plugins/jx-secret/pkg/schemas"
	"github.com/jenkins-x-plugins/jx-secret/pkg/schemas/testschemas"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadSurveySchema(t *testing.T) {
	fileName := filepath.Join("test_data", "load", "schema.yaml")
	s, err := schemas.LoadSchemaFile(fileName)
	assert.NoError(t, err, "should not have errored when loading survey schema")
	assert.Equal(t, 3, len(s.Spec.Objects), "should have matched 10 survey schema")

	property := testschemas.RequireSchemaProperty(t, s, "jx-admin-user", "username")
	assert.Equal(t, "admin", property.DefaultValue, "jx-admin-user.username.DefaultValue")

	property = testschemas.RequireSchemaProperty(t, s, "jx-pipeline-user", "token")
	assert.Equal(t, 40, property.MinLength, "jx-pipeline-user.token.MinLength")
	assert.Equal(t, 40, property.MaxLength, "jx-pipeline-user.token.MaxLength")
}

func TestValidateMissingName(t *testing.T) {
	fileName := filepath.Join("test_data", "validate", "schema.yaml")
	s, err := schemas.LoadSchemaFile(fileName)
	require.Error(t, err, "should have errored when validating survey s")
	t.Logf("got expected error %s", err.Error())
	assert.Contains(t, err.Error(), "Spec.Objects[0].Properties[1].Name: zero value", "should have failed validation")
	if s != nil {
		t.Logf("loaded s %#v", s.Spec)
	}
}
