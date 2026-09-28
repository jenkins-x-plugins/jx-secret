package extsecrets

import (
	"context"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
)

// StoreInterface fetches the (Cluster)SecretStore an ExternalSecret refers to.
type StoreInterface interface {
	// GetStore looks a namespaced SecretStore up in esNamespace. An empty kind
	// defaults to SecretStore, as ESO itself does.
	GetStore(kind, name, esNamespace string) (esv1.GenericStore, error)
}

type storeClient struct {
	dynamicClient dynamic.Interface
}

func NewStoreClient(dynClient dynamic.Interface) (StoreInterface, error) {
	dynClient, err := kube.LazyCreateDynamicClient(dynClient)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create a dynamic client")
	}
	return &storeClient{dynamicClient: dynClient}, nil
}

func (c *storeClient) GetStore(kind, name, esNamespace string) (esv1.GenericStore, error) {
	if name == "" {
		return nil, errors.New("no secretStoreRef name given")
	}

	var resource dynamic.ResourceInterface
	var store esv1.GenericStore
	if kind == esv1.ClusterSecretStoreKind {
		resource = c.dynamicClient.Resource(ClusterSecretStoresResource)
		store = &esv1.ClusterSecretStore{}
	} else {
		resource = c.dynamicClient.Resource(SecretStoresResource).Namespace(esNamespace)
		store = &esv1.SecretStore{}
	}

	u, err := resource.Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get %s %s", kind, name)
	}
	if err := FromUnstructured(u, store); err != nil {
		return nil, errors.Wrapf(err, "failed to convert to %s %s", kind, name)
	}
	return store, nil
}
