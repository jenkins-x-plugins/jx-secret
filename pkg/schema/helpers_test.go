package schema_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/jenkins-x/jx-secret/pkg/schema"
	"github.com/stretchr/testify/assert"
)

func TestLoadSurveySchema(t *testing.T) {
	fileName := filepath.Join("test_data", "load", "schema.yaml")
	surveySchema, err := schema.LoadSurveySchema(fileName)
	assert.NoError(t, err, "should not have errored when loading survey schema")
	assert.Equal(t, 10, len(surveySchema.Spec.Survey), "should have matched 10 survey schema")
}

func TestValidateMissingName(t *testing.T) {
	fileName := filepath.Join("test_data", "validate", "schema.yaml")
	_, err := schema.LoadSurveySchema(fileName)
	assert.Error(t, err, "should have errored when validating survey schema")
	assert.True(t, strings.Contains(err.Error(), "Spec.Survey[1].Name: zero value"), "should have failed validation")
}
