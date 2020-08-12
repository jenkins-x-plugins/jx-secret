package generators

import (
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GetSecretEntry returns a secret entry for a namespace, secret and secret entry
func GetSecretEntry(kubeClient kubernetes.Interface, namespace, secretName, entry string) (string, error) {
	if namespace == "" {
		log.Logger().Warnf("no namespace specified when trying to find secret %s entry %s", secretName, entry)
	}
	secret, err := kubeClient.CoreV1().Secrets(namespace).Get(secretName, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			log.Logger().Warnf("could not find secret %s with entry %s in namespace %s", secretName, entry, namespace)
			return "", nil
		}
		return "", errors.Wrapf(err, "failed to find Secret %s in namespace %s", secretName, namespace)
	}
	data := secret.Data
	if data != nil {
		return string(data[entry]), nil
	}
	return "", nil
}

// SecretEntry creates a generator for a secret
func SecretEntry(kubeClient kubernetes.Interface, namespace, secretName, entry string) Generator {
	return func(args *Arguments) (string, error) {
		return GetSecretEntry(kubeClient, namespace, secretName, entry)
	}
}
