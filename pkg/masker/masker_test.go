// +build unit

package masker_test

import (
	"testing"

	"github.com/jenkins-x-plugins/jx-secret/pkg/extsecrets"
	"github.com/jenkins-x-plugins/jx-secret/pkg/masker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func TestMasker(t *testing.T) {
	ns := "jx"

	testCases := []struct {
		name  string
		entry string
		hide  bool
		value string
	}{
		{
			name:  "knative-git-user-pass",
			entry: "username",
			hide:  false,
		},
		{
			name:  "knative-git-user-pass",
			entry: "password",
			hide:  true,
		},
		{
			name:  "random-secret",
			entry: "password",
			hide:  false,
		},
	}
	var k8sObjects []runtime.Object
	secrets := map[string]*corev1.Secret{}

	// lets populate the secrets with unique values for easier testing
	for i := range testCases {
		tc := &testCases[i]
		secretName := tc.name
		secret := secrets[secretName]
		if secret == nil {
			secret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:        secretName,
					Namespace:   ns,
					Annotations: map[string]string{},
				},
				Data: map[string][]byte{},
			}
			secrets[secretName] = secret
			k8sObjects = append(k8sObjects, secret)
		}
		entry := tc.entry
		tc.value = "ok-" + secretName + "-" + entry
		if tc.hide {
			tc.value = "HIDE-" + secretName + "-" + entry

			// add a dummy annotation for the schema
			secret.Annotations[extsecrets.SchemaObjectAnnotation] = `{"properties":[{"name":"token","question":"todo","help":"todo","generator":"gitOperator.password"}]}`
		}
		secret.Data[entry] = []byte(tc.value)
	}

	client := fake.NewSimpleClientset(k8sObjects...)

	m, err := masker.NewMasker(client, ns)
	require.NoError(t, err, "failed to create master")

	for _, tc := range testCases {
		// lets generate a text with each value
		rawText := "some random text [" + tc.value + "] more random text"

		masked := m.Mask(rawText)

		secretName := tc.name
		entry := tc.entry
		if tc.hide {
			if assert.NotEqual(t, rawText, masked, "should have masked the password value for secret %s entry %s", secretName, entry) {
				t.Logf("correctly masked the input %s => %s\n", rawText, masked)
			}
		} else {
			if assert.Equal(t, rawText, masked, "should not have masked the password value for secret %s entry %s", secretName, entry) {
				t.Logf("correctly did not mask: %s\n", rawText)
			}
		}
	}
}
