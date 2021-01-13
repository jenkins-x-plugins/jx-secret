package testsecrets

import (
	"k8s.io/apimachinery/pkg/runtime"
	dynfake "k8s.io/client-go/dynamic/fake"
)

// NewFakeDynClient creates a new dynamic client with the external secrets
func NewFakeDynClient(scheme *runtime.Scheme, dynObjects ...runtime.Object) *dynfake.FakeDynamicClient {
	/* TODO when we are on k8s 0.20
	gvrToListKind := map[schema.GroupVersionResource]string{
		{Group: "kubernetes-client.io", Version: "v1", Resource: "externalsecrets"}: "ExternalSecretList",
	}
	return dynfake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrToListKind, dynObjects...)
	*/
	return dynfake.NewSimpleDynamicClient(scheme, dynObjects...)
}
