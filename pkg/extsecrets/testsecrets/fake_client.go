package testsecrets

import (
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/jenkins-x-plugins/jx-secret/pkg/extsecrets"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynfake "k8s.io/client-go/dynamic/fake"
)

// NewFakeDynClient creates a new dynamic client with the external secrets
func NewFakeDynClient(scheme *runtime.Scheme, dynObjects ...runtime.Object) *dynfake.FakeDynamicClient {
	gvrToListKind := map[schema.GroupVersionResource]string{
		extsecrets.ExternalSecretsResource:     esv1.ExtSecretKind + "List",
		extsecrets.SecretStoresResource:        esv1.SecretStoreKind + "List",
		extsecrets.ClusterSecretStoresResource: esv1.ClusterSecretStoreKind + "List",
	}
	return dynfake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrToListKind, dynObjects...)
}
