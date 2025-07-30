package extsecrets

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx-helpers/v3/pkg/kube"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

var (
	// ExternalSecretsResource the schema group version resource
	KubernetesExternalSecretsResource         = schema.GroupVersionResource{Group: "kubernetes-client.io", Version: "v1", Resource: "externalsecrets"}
	ExternalSecretsResource = schema.GroupVersionResource{Group: "external-secrets.io", Version: "v1beta1", Resource: "externalsecrets"}

	info = termcolor.ColorInfo
)

// NewClient creates a new client from the given dynamic client
func NewClient(dynClient dynamic.Interface, kubeClient kubernetes.Interface) (Interface, error) {
	var err error
	dynClient, err = kube.LazyCreateDynamicClient(dynClient)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create a dynamic client")
	}
	kubeClient, err = kube.LazyCreateKubeClient(kubeClient)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create a kubernetes client")
	}
	return &client{dynamicClient: dynClient, kubeClient: kubeClient}, nil
}

// SimplifyKey simplify the key to avoid unnecessary paths
func SimplifyKey(backendType, key string) string {
	if backendType != "vault" {
		return key
	}

	// we shouldn't pass in secret/data/foo when using the CLI tool
	if strings.HasPrefix(key, "secret/data/") {
		key = "secret/" + strings.TrimPrefix(key, "secret/data/")
	}
	return key
}

// CopySecretToNamespace copies the given secret to the namespace
func CopySecretToNamespace(kubeClient kubernetes.Interface, ns string, fromSecret *corev1.Secret) error {
	secretInterface := kubeClient.CoreV1().Secrets(ns)
	name := fromSecret.Name
	secret, err := secretInterface.Get(context.TODO(), name, metav1.GetOptions{})

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
		_, err = secretInterface.Create(context.TODO(), secret, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrapf(err, "failed to create Secret %s in namespace %s", name, ns)
		}
		log.Logger().Infof("created Secret %s in namespace %s", info(name), info(ns))
		return nil
	}
	_, err = secretInterface.Update(context.TODO(), secret, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to update Secret %s in namespace %s", name, ns)
	}
	log.Logger().Infof("updated Secret %s in namespace %s", info(name), info(ns))
	return nil
}

// DefaultHelmSecretFolder creates a default helm secret folder
func DefaultHelmSecretFolder() string {
	answer := os.Getenv("JX_HELM_SECRET_FOLDER")
	if answer == "" {
		answer = filepath.Join(os.TempDir(), "secrets", "jx-helm")
	}
	return answer
}
