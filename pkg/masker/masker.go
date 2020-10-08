package masker

import (
	"context"
	"strings"

	"github.com/jenkins-x/jx-helpers/v3/pkg/stringhelpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
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
)

// Client replaces words in a log from a set of secrets
type Client struct {
	ReplaceWords map[string]string
}

// NewMasker creates a new Client loading secrets from the given namespace
func NewMasker(kubeClient kubernetes.Interface, ns string) (*Client, error) {
	masker := &Client{}
	ctx := context.Background()
	resourceList, err := kubeClient.CoreV1().Secrets(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return masker, err
	}
	for _, s := range resourceList.Items {
		secret := s
		masker.LoadSecret(&secret)
	}
	return masker, nil
}

// LoadSecrets loads the secrets into the log masker
func (m *Client) LoadSecrets(kubeClient kubernetes.Interface, ns string) error {
	ctx := context.Background()
	resourceList, err := kubeClient.CoreV1().Secrets(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, s := range resourceList.Items {
		secret := s
		err = m.LoadSecret(&secret)
		if err != nil {
			return errors.Wrapf(err, "failed to load secret %s in namespace %s", s.Name, ns)
		}
	}
	return nil
}

// LoadSecret loads the secret data into the log masker
func (m *Client) LoadSecret(secret *corev1.Secret) error {
	if m.ReplaceWords == nil {
		m.ReplaceWords = map[string]string{}
	}

	secretName := secret.Name
	if stringhelpers.StringArrayIndex(ignoreSecrets, secretName) >= 0 {
		log.Logger().Infof("ignoring secret %s", info(secretName))
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
		return nil
	}

	count := 0
	for i := range schemaObject.Properties {
		prop := schemaObject.Properties[i]
		if prop.Template != "" {
			return nil
		}
		count++
	}

	if count > 0 {
		ignoredProperties := ignoreSecretProperties[secretName]

		for name, d := range secret.Data {
			if ignoredProperties != nil && ignoredProperties[name] {
				log.Logger().Infof("ignoring secret %s entry %s", info(secretName), info(name))
				continue
			}
			value := string(d)
			if len(value) < 5 {
				// assume dummy value
				continue
			}

			if m.ReplaceWords[value] == "" {
				password := ""
				if ShowMaskedPasswords {
					password = " => " + value
				}
				log.Logger().Infof("adding mask of secret %s entry %s %s", info(secretName), info(name), password)
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

// replaceMapValues adds all the string values in the given map to the replacer words
func (m *Client) replaceMapValues(values map[string]interface{}) {
	for _, value := range values {
		childMap, ok := value.(map[string]interface{})
		if ok {
			m.replaceMapValues(childMap)
			continue
		}
		text, ok := value.(string)
		if ok {
			m.ReplaceWords[text] = m.replaceValue(text)
		}
	}
}

func (m *Client) replaceValue(_ string) string {
	return MaskedOut
}
