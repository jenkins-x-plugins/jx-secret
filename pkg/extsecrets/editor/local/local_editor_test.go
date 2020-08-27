package local_test

import (
	"testing"

	"github.com/jenkins-x/jx-helpers/pkg/testhelpers"
	v1 "github.com/jenkins-x/jx-secret/pkg/apis/external/v1"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets/editor"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets/editor/local"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestLocalEditor(t *testing.T) {
	kubeClient := fake.NewSimpleClientset()

	ns := "jx"
	secretName := "knative-docker-user-pass"
	propertyName := "token"
	expectedPropertyValue := "someTokenValue"

	extSecret := &v1.ExternalSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: ns,
		},
		Spec:   v1.ExternalSecretSpec{},
		Status: nil,
	}
	secretEditor, err := local.NewEditor(kubeClient, extSecret)
	require.NoError(t, err, "failed to create editor")

	properties := &editor.KeyProperties{
		Properties: []editor.PropertyValue{
			{
				Property: propertyName,
				Value:    expectedPropertyValue,
			},
		},
	}
	err = secretEditor.Write(properties)
	require.NoError(t, err, "failed to write properties")

	secret, message := testhelpers.RequireSecretExists(t, kubeClient, ns, secretName)
	testhelpers.AssertSecretEntryEquals(t, secret, propertyName, expectedPropertyValue, message)
}
