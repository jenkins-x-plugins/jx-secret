package verify_test

import (
	"testing"

	"github.com/jenkins-x/jx-secret/pkg/cmd/verify"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets/testsecrets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
)

func TestVerify(t *testing.T) {
	var err error
	_, o := verify.NewCmdVerify()
	scheme := runtime.NewScheme()

	ns := "jx"

	kubeObjects := []runtime.Object{
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "knative-docker-user-pass",
				Namespace: ns,
			},
			Data: map[string][]byte{
				"username": []byte("dummyValue"),
			},
		},
	}
	dynObjects := testsecrets.LoadExtSecretDir(t, ns, "test_data")

	gvrToListKind := map[schema.GroupVersionResource]string{
		{Group: "kubernetes-client.io", Version: "v1", Resource: "externalsecrets"}: "ExternalSecretList",
	}
	fakeDynClient := dynfake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrToListKind, dynObjects...)
	o.SecretClient, err = extsecrets.NewClient(fakeDynClient)
	require.NoError(t, err, "failed to create fake extsecrets Client")

	o.Namespace = ns
	o.KubeClient = fake.NewSimpleClientset(kubeObjects...)

	err = o.Run()
	require.NoError(t, err, "failed to run verify")

	assert.Len(t, o.Results, 2, "results")

	for _, r := range o.Results {
		name := r.ExternalSecret.Name
		assert.Len(t, r.EntryErrors, 1, "error count for %s", name)

		for _, e := range r.EntryErrors {
			switch name {
			case "knative-docker-user-pass":
				assert.Equal(t, []string{"password"}, e.Properties, "missing properties for %s", name)
			case "lighthouse-oauth-token":
				assert.Equal(t, []string{"token"}, e.Properties, "missing properties for %s", name)
			default:
				assert.Fail(t, "unknown secret name %s", name)
			}
		}
	}
}
