package gsm

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx-helpers/v3/pkg/testhelpers"

	"github.com/stretchr/testify/assert"

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
	existingSecrets := make(map[string]string)
	existingSecrets["fish"] = "chips"
	err = c.writeTemporarySecretPropertiesJSON(existingSecrets, p, file)
	assert.NoError(t, err)

	testhelpers.AssertTextFileContentsEqual(t, filepath.Join("test_data", "write_secrets", "expected.json"), file.Name())
}

func Test_client_WriteSingle(t *testing.T) {

	file, err := ioutil.TempFile("", "jx")
	assert.NoError(t, err, "should not error creating a temporary file")

	c := &client{}

	p := &editor.KeyProperties{
		Properties: []editor.PropertyValue{
			{
				Value: "bar",
			},
		},
	}
	existingSecrets := make(map[string]string)
	err = c.writeTemporarySecretPropertiesJSON(existingSecrets, p, file)
	assert.NoError(t, err)

	testhelpers.AssertTextFileContentsEqual(t, filepath.Join("test_data", "write_single", "expected.json"), file.Name())
}

func Test_client_WriteMultiline(t *testing.T) {

	file, err := ioutil.TempFile("", "jx")
	assert.NoError(t, err, "should not error creating a temporary file")

	c := &client{}

	p := &editor.KeyProperties{
		Properties: []editor.PropertyValue{
			{
				Property: "foo",
				Value:    "-----BEGIN PUBLIC KEY-----\nabc123\n123abc==\n-----END PUBLIC KEY-----",
			},
		},
	}
	existingSecrets := make(map[string]string)
	existingSecrets["foo"] = "chips"
	err = c.writeTemporarySecretPropertiesJSON(existingSecrets, p, file)
	assert.NoError(t, err)

	testhelpers.AssertTextFileContentsEqual(t, filepath.Join("test_data", "write_multiline", "expected.json"), file.Name())

}
