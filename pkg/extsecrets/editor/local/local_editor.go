package local

import (
	"strings"

	"github.com/jenkins-x/jx-helpers/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/pkg/log"
	v1 "github.com/jenkins-x/jx-secret/pkg/apis/external/v1"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets"
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
	extsec     *v1.ExternalSecret
}

func NewEditor(kubeClient kubernetes.Interface, extsec *v1.ExternalSecret) (editor.Interface, error) {
	if extsec.Name == "" {
		return nil, errors.Errorf("missing ExternalSecret.name")
	}
	if extsec.Namespace == "" {
		return nil, errors.Errorf("missing ExternalSecret.namespace for external secret %s", extsec.Name)
	}

	c := &client{
		kubeClient: kubeClient,
		extsec:     extsec,
	}
	return c, nil
}

// Write writes the properties to the Secret
func (c *client) Write(properties *editor.KeyProperties) error {
	create := false
	extsec := c.extsec
	name := extsec.Name
	ns := extsec.Namespace
	typeName := extsec.Spec.Template.Type
	if typeName == "" {
		typeName = string(corev1.SecretTypeOpaque)
	}

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
	secret.Type = corev1.SecretType(typeName)
	if secret.Data == nil {
		secret.Data = map[string][]byte{}
	}

	for _, pv := range properties.Properties {
		secret.Data[pv.Property] = []byte(pv.Value)
	}

	// lets copy any annotations from the template
	md := extsec.Spec.Template.Metadata
	if md.Annotations != nil {
		if secret.Annotations == nil {
			secret.Annotations = map[string]string{}
		}
		for k, v := range md.Annotations {
			secret.Annotations[k] = v
		}
	}
	if md.Labels != nil {
		if secret.Labels == nil {
			secret.Labels = map[string]string{}
		}
		for k, v := range md.Labels {
			secret.Labels[k] = v
		}
	}

	if create {
		_, err = secretInterface.Create(secret)
		if err != nil {
			return errors.Wrapf(err, "failed to create Secret %s in namespace %s", name, ns)
		}
		log.Logger().Infof("created Secret %s in namespace %s", info(name), info(ns))
	} else {
		_, err = secretInterface.Update(secret)
		if err != nil {
			return errors.Wrapf(err, "failed to update Secret %s in namespace %s", name, ns)
		}
		log.Logger().Infof("updated Secret %s in namespace %s", info(name), info(ns))
	}

	// lets check for replicated secrets
	if extsec.Annotations != nil {
		namespaces := extsec.Annotations[extsecrets.ReplicateAnnotation]
		if namespaces != "" {
			nsList := strings.Split(namespaces, ",")
			for _, tons := range nsList {
				err = c.replicateSecretToNamespace(secret, tons)
				if err != nil {
					return errors.Wrapf(err, "failed to replicate Secret for local backend")
				}
			}
		}
	}
	return nil
}

func (c *client) replicateSecretToNamespace(fromSecret *corev1.Secret, ns string) error {
	secretInterface := c.kubeClient.CoreV1().Secrets(ns)
	name := fromSecret.Name
	secret, err := secretInterface.Get(name, metav1.GetOptions{})

	create := false
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
		}
	}

	if string(fromSecret.Type) != "" {
		secret.Type = fromSecret.Type
	}
	if fromSecret.Annotations != nil {
		if secret.Annotations == nil {
			secret.Annotations = map[string]string{}
		}
		for k, v := range fromSecret.Annotations {
			secret.Annotations[k] = v
		}
	}

	if fromSecret.Labels != nil {
		if secret.Labels == nil {
			secret.Labels = map[string]string{}
		}
		for k, v := range fromSecret.Labels {
			secret.Labels[k] = v
		}
	}
	if fromSecret.Data != nil {
		if secret.Data == nil {
			secret.Data = map[string][]byte{}
		}
		for k, v := range fromSecret.Data {
			secret.Data[k] = v
		}
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
