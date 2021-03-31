package azure

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/v7.1/keyvault"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/jenkins-x-plugins/jx-secret/pkg/extsecrets/editor"
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

func NewEditor(keyVaultName string, keyVaultClient KeyVault) (editor.Interface, error) {
	c := &client{
		keyVaultClient: keyVaultClient,
		keyVaultUrl:    fmt.Sprintf("https://%s.vault.azure.net/", keyVaultName),
	}
	return c, nil
}

func (c *client) Write(properties *editor.KeyProperties) error {

	secretValue, err := formatSecretValue(properties.Properties)
	if err != nil {
		return errors.Wrap(err, "error formatting secret value for Azure Key Vault")
	}

	err = c.keyVaultClient.SetSecret(c.keyVaultUrl, properties.Key, secretValue)
	if err != nil {
		return errors.Wrapf(err, "error setting azure key vault secret")
	}

	return nil
}

func formatSecretValue(propertyValues []editor.PropertyValue) (string, error) {

	if len(propertyValues) == 0 {
		return "", fmt.Errorf("no values found for secret")
	} else if len(propertyValues) == 1 {
		return propertyValues[0].Value, nil
	}

	propVals := map[string]string{}

	for _, prop := range propertyValues {
		propVals[prop.Property] = prop.Value
	}

	propValString, err := json.Marshal(propVals)

	if err != nil {
		return "", errors.Wrap(err, "error serializing complex secret type to json")
	}

	return string(propValString), nil

}
