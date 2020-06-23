package extsecrets

import (
	"strings"

	"github.com/jenkins-x/jx-kube-client/pkg/kubeclient"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

var (
	ExternalSecretsResource = schema.GroupVersionResource{Group: "kubernetes-client.io", Version: "v1", Resource: "externalsecrets"}
)

// NewClient creates a new client from the given dynamic client
func NewClient(dynClient dynamic.Interface) (Interface, error) {
	var err error
	dynClient, err = LazyCreateDynamicClient(dynClient)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create a dynamic client")
	}

	return &client{dynamicClient: dynClient}, nil
}

// LazyCreateDynamicClient lazily creates the dynamic client if its not defined
func LazyCreateDynamicClient(client dynamic.Interface) (dynamic.Interface, error) {
	if client != nil {
		return client, nil
	}
	f := kubeclient.NewFactory()
	cfg, err := f.CreateKubeConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kubernetes config")
	}
	client, err = dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "error building dynamic clientset")
	}
	return client, nil
}

// LazyCreateKubeClient lazy creates the kube client if its not defined
func LazyCreateKubeClient(client kubernetes.Interface) (kubernetes.Interface, error) {
	if client != nil {
		return client, nil
	}
	f := kubeclient.NewFactory()
	cfg, err := f.CreateKubeConfig()
	if err != nil {
		return client, errors.Wrap(err, "failed to get kubernetes config")
	}
	client, err = kubernetes.NewForConfig(cfg)
	if err != nil {
		return client, errors.Wrap(err, "error building kubernetes clientset")
	}
	return client, nil
}

// SimplifyKey simplify the key to avoid unnecessary paths
func SimplifyKey(backendType string, key string) string {
	if backendType != "vault" {
		return key
	}

	// we shouldn't pass in secret/data/foo when using the CLI tool
	if strings.HasPrefix(key, "secret/data/") {
		key = "secret/" + strings.TrimPrefix(key, "secret/data/")
	}
	return key
}
