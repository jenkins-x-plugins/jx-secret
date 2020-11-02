package masker

import (
	"context"
	"fmt"
	"strings"

	"github.com/jenkins-x/jx-helpers/v3/pkg/stringhelpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/jenkins-x/jx-secret/pkg/apis/schema/v1alpha1"
	"github.com/jenkins-x/jx-secret/pkg/schemas"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// MaskedOut the text that is returned if text is masked out
	// we don't use the length of the actual secret value to avoid giving folks a hint what the value is
	MaskedOut = "****"
)

var (
	// ShowMaskedPasswords lets you enable in test programs
	ShowMaskedPasswords = false

	info = termcolor.ColorInfo

	// ignoreSecrets default secrets to ignore
	//
	// TODO add a flag on the schema for this?
	ignoreSecrets = []string{"bucketrepo-config", "knative-docker-user-pass"}

	ignoreSecretProperties = map[string]map[string]bool{
		"knative-git-user-pass": {
			"username": true,
		},
		"jenkins-x-bucketrepo": {
			"BASIC_AUTH_USER": true,
		},
		"jenkins-x-chartmuseum": {
			"BASIC_AUTH_USER": true,
		},
	}

	// customSchemaObjects additional schemas if they are missing from Secrets
	customSchemaObjects = map[string]*v1alpha1.Object{
		"jx-boot": {
			Properties: []v1alpha1.Property{
				{
					Name: "password",
				},
				{
					Name:   "url",
					NoMask: true,
				},
				{
					Name:   "username",
					NoMask: true,
				},
			},
		},
	}
)

// Client replaces words in a log from a set of secrets
type Client struct {
	ReplaceWords map[string]string
	LogFn        func(string)
}

// NewMasker creates a new Client loading secrets from the given namespace
func NewMasker(kubeClient kubernetes.Interface, namespaces ...string) (*Client, error) {
	masker := &Client{}
	ctx := context.Background()

	for _, ns := range namespaces {
		resourceList, err := kubeClient.CoreV1().Secrets(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			return masker, err
		}
		for i := range resourceList.Items {
			secret := &resourceList.Items[i]
			err = masker.LoadSecret(secret)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to load secret %s", secret.Name)
			}
		}
	}
	return masker, nil
}

// GetReplacedWords returns all the words that will be replaced
func (m *Client) GetReplacedWords() []string {
	var words []string
	for k := range m.ReplaceWords {
		words = append(words, k)
	}
	return words
}

// LoadSecrets loads the secrets into the log masker
func (m *Client) LoadSecrets(kubeClient kubernetes.Interface, ns string) error {
	ctx := context.Background()
	resourceList, err := kubeClient.CoreV1().Secrets(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	for i := range resourceList.Items {
		secret := &resourceList.Items[i]
		err = m.LoadSecret(secret)
		if err != nil {
			return errors.Wrapf(err, "failed to load secret %s in namespace %s", secret.Name, ns)
		}
	}
	return nil
}

// LoadSecret loads the secret data into the log masker
func (m *Client) LoadSecret(secret *corev1.Secret) error {
	if m.ReplaceWords == nil {
		m.ReplaceWords = map[string]string{}
	}

	if m.LogFn == nil {
		m.LogFn = func(text string) {
			log.Logger().Info(text)
		}
	}
	secretName := secret.Name
	if stringhelpers.StringArrayIndex(ignoreSecrets, secretName) >= 0 {
		m.LogFn(fmt.Sprintf("ignoring secret %s", info(secretName)))
		return nil
	}
	if len(secret.Data) == 0 {
		return nil
	}
	schemaObject, err := schemas.ObjectFromObjectMeta(&secret.ObjectMeta)
	if err != nil {
		return errors.Wrapf(err, "failed to get jx-secret Schema object")
	}

	// lets ignore secrets without schemas
	if schemaObject == nil {
		schemaObject = customSchemaObjects[secretName]
	}
	if schemaObject == nil {
		return nil
	}

	ignoredProperties := map[string]bool{}
	for k, v := range ignoreSecretProperties[secretName] {
		ignoredProperties[k] = v
	}
	for i := range schemaObject.Properties {
		p := &schemaObject.Properties[i]
		if p.NoMask {
			ignoredProperties[p.Name] = true
		}
	}
	if len(schemaObject.Properties) > 0 {
		for name, d := range secret.Data {
			if ignoredProperties[name] {
				m.LogFn(fmt.Sprintf("ignoring secret %s entry %s", info(secretName), info(name)))
				continue
			}
			value := string(d)
			if len(value) < 5 {
				// assume dummy value
				continue
			}

			if m.ReplaceWords[value] == "" {
				password := "" //NOSONAR
				if ShowMaskedPasswords {
					password = " => " + value
				}
				m.LogFn(fmt.Sprintf("adding mask of secret %s entry %s %s", info(secretName), info(name), password))
				m.ReplaceWords[value] = m.replaceValue(value)
			}
		}
	}
	return nil
}

// Mask returns the text with all of the secrets masked out
func (m *Client) Mask(text string) string {
	answer := text
	for k, v := range m.ReplaceWords {
		answer = strings.Replace(answer, k, v, -1)
	}
	return answer
}

// MaskData masks the given data
func (m *Client) MaskData(logData []byte) []byte {
	text := m.Mask(string(logData))
	return []byte(text)
}

func (m *Client) replaceValue(_ string) string {
	return MaskedOut
}
