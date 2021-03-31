package gsm

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/alecthomas/assert"

	"github.com/jenkins-x-plugins/jx-secret/pkg/extsecrets/editor"
)

func Test_client_Write(t *testing.T) {

	file, err := ioutil.TempFile("", "jx")
	assert.NoError(t, err, "should not error creating a temporary file")

	c := &client{}

	p := &editor.KeyProperties{
		Properties: []editor.PropertyValue{
			{
				Property: "foo",
				Value:    "bar",
			},
			{
				Property: "wine",
				Value:    "cheese",
			},
		},
	}
	err = c.writeTemporarySecretPropertiesJSON(p, file)
	assert.NoError(t, err)

	expected, err := ioutil.ReadFile(filepath.Join("test_data", "write_secrets", "expected.json"))
	assert.NoError(t, err, "shouldn't fail to read expected results file")

	results, err := ioutil.ReadFile(file.Name())
	assert.NoError(t, err, "shouldn't fail to read results file")

	assert.Equal(t, expected, results, "json file should contain key/value pairs of secret properties")
}
