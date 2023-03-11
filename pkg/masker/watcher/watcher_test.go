//go:build unit
// +build unit

package watcher_test

import (
	"strings"
	"testing"

	"github.com/jenkins-x-plugins/jx-secret/pkg/extsecrets"
	"github.com/jenkins-x-plugins/jx-secret/pkg/masker/watcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestWatcherMasker(t *testing.T) {
	ns := "jx"

	kubeClient := fake.NewSimpleClientset()

	o := &watcher.Options{
		Namespaces: []string{"jx", "jx-git-operator"},
		KubeClient: kubeClient,
	}
	err := o.Validate()
	require.NoError(t, err, "failed to validate")

	const userPasswordSchemaAnnotation = `{"properties":[{"name":"username","noMask":true}]}`

	testCases := []struct {
		secret             *corev1.Secret
		expectedWords      []string
		shouldNotHaveWords []string
		namespace          string
	}{
		{
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret1",
					Namespace: ns,
					Annotations: map[string]string{
						extsecrets.SchemaObjectAnnotation: userPasswordSchemaAnnotation,
					},
				},
				Data: map[string][]byte{
					"username": []byte("myuser"),
					"password": []byte("mypassword"),
				},
			},
			expectedWords:      []string{"mypassword"},
			shouldNotHaveWords: []string{"myuser"},
		},
		{
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ignored",
					Namespace: ns,
				},
				Data: map[string][]byte{
					"username": []byte("dummyuser"),
					"password": []byte("dummypassword"),
				},
			},
			expectedWords: []string{"mypassword"},
		},
		{
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret1-updated",
					Namespace: ns,
					Annotations: map[string]string{
						extsecrets.SchemaObjectAnnotation: userPasswordSchemaAnnotation,
					},
				},
				Data: map[string][]byte{
					"username": []byte("myuser"),
					"password": []byte("mynewpassword"),
				},
			},
			expectedWords:      []string{"mypassword", "mynewpassword"},
			shouldNotHaveWords: []string{"myuser"},
		},
	}

	// lets populate the secrets with unique values for easier testing
	for i := range testCases {
		tc := &testCases[i]

		name := tc.secret.Name
		if tc.namespace == "" {
			tc.namespace = ns
		}

		o.UpsertSecret(tc.namespace, tc.secret)

		client := o.GetClient()
		require.NotNil(t, client, "no client for test %s", name)

		for _, w := range tc.expectedWords {
			if client.ReplaceWords[w] == "" {
				assert.Failf(t, "excluded secret", "should replace word %s for test %s", w, name)
			}
		}
		for _, w := range tc.shouldNotHaveWords {
			if client.ReplaceWords[w] != "" {
				assert.Failf(t, "included secret", "should not have replace word %s for test %s", w, name)
			}
		}

		t.Logf("after secret %s replace words are: %s\n", name, strings.Join(client.GetReplacedWords(), ","))
	}
}
