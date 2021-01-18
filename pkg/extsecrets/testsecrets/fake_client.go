package testsecrets

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynfake "k8s.io/client-go/dynamic/fake"
)

// NewFakeDynClient creates a new dynamic client with the external secrets
func NewFakeDynClient(scheme *runtime.Scheme, dynObjects ...runtime.Object) *dynfake.FakeDynamicClient {
	gvrToListKind := map[schema.GroupVersionResource]string{
		{Group: "kubernetes-client.io", Version: "v1", Resource: "externalsecrets"}: "ExternalSecretList",
	}
	return dynfake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrToListKind, dynObjects...)
}
