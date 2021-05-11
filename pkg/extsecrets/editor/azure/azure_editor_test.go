package azure_test

import (
	"testing"

	editor2 "github.com/jenkins-x-plugins/jx-secret/pkg/extsecrets/editor"
	"github.com/jenkins-x-plugins/jx-secret/pkg/extsecrets/editor/azure"
	"github.com/stretchr/testify/assert"
)

type Mock struct{}

func (m Mock) SetSecret(vaultBaseURL string, secretName string, secretValue string) error {
	return nil
}

func TestAzureKeyVaultSingleValue(t *testing.T) {
	m := Mock{}
	editor, err := azure.NewEditor("vaultUrl", m)
	assert.NoError(t, err)
	err = editor.Write(&editor2.KeyProperties{
		Key: "keyName",
		Properties: []editor2.PropertyValue{
			{
				Value: "flameproofboots",
			},
		},
	})
	assert.NoError(t, err)
}

func TestAzureKeyVaultNoValues(t *testing.T) {
	m := Mock{}
	editor, err := azure.NewEditor("vaultUrl", m)
	assert.NoError(t, err)
	err = editor.Write(&editor2.KeyProperties{
		Key:        "keyName",
		Properties: []editor2.PropertyValue{},
	})
	assert.Error(t, err)
}

func TestAzureKeyVaultSettingPropertyName(t *testing.T) {
	m := Mock{}
	editor, err := azure.NewEditor("vaultUrl", m)
	assert.NoError(t, err)
	err = editor.Write(&editor2.KeyProperties{
		Key: "keyName",
		Properties: []editor2.PropertyValue{
			{
				Property: "pinkiepie",
				Value:    "rollerdisco",
			},
		},
	})
	assert.NoError(t, err)
}
