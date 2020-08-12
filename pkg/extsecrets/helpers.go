package extsecrets

import (
	"strings"

	"github.com/jenkins-x/jx-helpers/pkg/kube"
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
	dynClient, err = kube.LazyCreateDynamicClient(dynClient)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create a dynamic client")
	}

	return &client{dynamicClient: dynClient}, nil
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
