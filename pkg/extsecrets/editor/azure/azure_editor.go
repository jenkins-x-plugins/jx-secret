package azure

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/v7.1/keyvault"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets/editor"
	"github.com/pkg/errors"
)

type client struct {
	keyVaultClient KeyVault
	keyVaultUrl    string
}

type KeyVault interface {
	SetSecret(vaultBaseURL string, secretName string, secretValue string) error
}

type KeyVaultClient struct{}

func (k KeyVaultClient) SetSecret(vaultBaseURL string, secretName string, secretValue string) error {

	client := keyvault.BaseClient{}
	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		return errors.Wrapf(err, "error creating authorizer from environment")
	}
	client.Authorizer = authorizer

	secretParams := keyvault.SecretSetParameters{
		Value: &secretValue,
	}

	_, err = client.SetSecret(context.TODO(), vaultBaseURL, secretName, secretParams)
	if err != nil {
		return errors.Wrapf(err, "error retrieving secret from key vault")
	}

	return nil
}

func NewEditor(keyVaultUrl string, keyVaultClient KeyVault) (editor.Interface, error) {
	c := &client{
		keyVaultClient: keyVaultClient,
		keyVaultUrl:    keyVaultUrl,
	}
	return c, nil
}

func (c *client) Write(properties *editor.KeyProperties) error {

	if len(properties.Properties) != 1 {
		return fmt.Errorf("more than one secret value specified which is not currently supported")
	}

	if properties.Properties[0].Property != "" {
		return fmt.Errorf("property name is specified which implies complex structure which is not currently supported")
	}

	if properties.Properties[0].Value == "" {
		return fmt.Errorf("property value is empty")
	}

	err := c.keyVaultClient.SetSecret(c.keyVaultUrl, properties.Key, properties.Properties[0].Value)
	if err != nil {
		return errors.Wrapf(err, "error setting azure key vault secret")
	}

	return nil
}
