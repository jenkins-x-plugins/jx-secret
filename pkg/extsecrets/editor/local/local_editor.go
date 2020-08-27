package local

import (
	"github.com/jenkins-x/jx-helpers/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/pkg/log"
	v1 "github.com/jenkins-x/jx-secret/pkg/apis/external/v1"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets/editor"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	info = termcolor.ColorInfo
)

type client struct {
	kubeClient kubernetes.Interface
	namespace  string
	name       string
	typeName   string
}

func NewEditor(kubeClient kubernetes.Interface, secret *v1.ExternalSecret) (editor.Interface, error) {
	name := secret.Name
	namespace := secret.Namespace
	if name == "" {
		return nil, errors.Errorf("missing ExternalSecret.name")
	}
	if namespace == "" {
		return nil, errors.Errorf("missing ExternalSecret.namespace for external secret %s", name)
	}

	typeName := secret.Spec.Template.Type
	if typeName == "" {
		typeName = string(corev1.SecretTypeOpaque)
	}

	c := &client{
		kubeClient: kubeClient,
		namespace:  namespace,
		name:       name,
		typeName:   typeName,
	}
	return c, nil
}

// Write writes the properties to the Secret
func (c *client) Write(properties *editor.KeyProperties) error {
	create := false
	name := c.name
	ns := c.namespace
	secretInterface := c.kubeClient.CoreV1().Secrets(ns)
	secret, err := secretInterface.Get(name, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return errors.Wrapf(err, "failed to ")
		}
		create = true
		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns,
			},
			Type: corev1.SecretTypeOpaque,
		}
	}
	secret.Type = corev1.SecretType(c.typeName)
	if secret.Data == nil {
		secret.Data = map[string][]byte{}
	}

	for _, pv := range properties.Properties {
		secret.Data[pv.Property] = []byte(pv.Value)
	}

	if create {
		_, err = secretInterface.Create(secret)
		if err != nil {
			return errors.Wrapf(err, "failed to create Secret %s in namespace %s", name, ns)
		}
		log.Logger().Infof("created Secret %s in namespace %s", info(name), info(ns))
		return nil
	}
	_, err = secretInterface.Update(secret)
	if err != nil {
		return errors.Wrapf(err, "failed to update Secret %s in namespace %s", name, ns)
	}
	log.Logger().Infof("updated Secret %s in namespace %s", info(name), info(ns))
	return nil
}
