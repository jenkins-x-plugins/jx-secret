package postrender_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/jenkins-x-plugins/jx-secret/pkg/cmd/postrender"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

var (
	// generateTestOutput enable to regenerate the expected output
	generateTestOutput = false
)

func TestPostrendererConvert(t *testing.T) {
	sourceFile := filepath.Join("test_data", "input.yaml")
	expectedFile := filepath.Join("test_data", "expected.yaml")

	data, err := ioutil.ReadFile(sourceFile)
	require.NoError(t, err, "failed to read %s", sourceFile)

	_, o := postrender.NewCmdPostrender()

	err = o.ConvertOptions.Validate()
	require.NoError(t, err, "failed validate options")

	if o.ConvertOptions.DefaultNamespace == "" {
		o.ConvertOptions.DefaultNamespace = "jx"
	}

	input := string(data)
	got, err := o.Convert(input)
	require.NoError(t, err, "failed to convert input")

	if generateTestOutput {
		dir := filepath.Dir(expectedFile)
		err := os.MkdirAll(dir, files.DefaultDirWritePermissions)
		require.NoError(t, err, "failed to make dir %s", dir)

		err = ioutil.WriteFile(expectedFile, []byte(got), 0666)
		require.NoError(t, err, "failed to save file %s", expectedFile)
		t.Logf("saved %s\n", expectedFile)
	} else {
		data, err := ioutil.ReadFile(expectedFile)
		require.NoError(t, err, "failed to read %s", expectedFile)

		assert.Equal(t, string(data), got, "expected output")
	}

	// lets verify the secret data....
	assert.NotEmpty(t, o.PopulateOptions.HelmSecretValues, "should have helm secret values")

	for k, m := range o.PopulateOptions.HelmSecretValues {
		t.Logf("%s = %#v\n", k, m)
	}
	key := "jx/mysecret"
	values := o.PopulateOptions.HelmSecretValues[key]
	assert.NotEmpty(t, values, "should have secret values for key %s", key)
	assert.Equal(t, "edam", values["cheese"], "no secret value for cheese with key %s", key)
	t.Logf("key %s cheese = %s\n", key, values["cheese"])
}
