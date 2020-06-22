package extsecrets

import (
	"github.com/jenkins-x/jx-kube-client/pkg/kubeclient"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
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
