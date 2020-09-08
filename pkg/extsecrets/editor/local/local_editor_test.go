package local_test

import (
	"testing"

	"github.com/jenkins-x/jx-helpers/pkg/testhelpers"
	v1 "github.com/jenkins-x/jx-secret/pkg/apis/external/v1"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets/editor"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets/editor/local"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestLocalEditor(t *testing.T) {
	kubeClient := fake.NewSimpleClientset()

	ns := "jx"
	/* #nosec */
	secretName := "knative-docker-user-pass"
	propertyName := "token"
	expectedPropertyValue := "someTokenValue"

	labelName := "mylabel"
	expetedLabelValue := "myLabelValue"

	annotationName := "myannotation"
	expectedAnnotationValue := "myAnnotationValue"

	expectedSchemaAnnotation := `{"name":"` + secretName + `","properties":[{"name":"username","question":"bucket repository user name","help":"The username used to authenticate with the bucket repository","defaultValue":"admin"},{"name":"password","question":"bucket repository password","help":"The password to authenticate with the bucket repository","minLength":5,"maxLength":41,"generator":"password"}]}`

	extSecret := &v1.ExternalSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: ns,
			Annotations: map[string]string{
				extsecrets.ReplicateToAnnotation: "jx-staging,jx-production",
			},
		},
		Spec: v1.ExternalSecretSpec{
			Template: v1.Template{
				Type: "MyType",
				Metadata: metav1.ObjectMeta{
					Labels: map[string]string{
						labelName: expetedLabelValue,
					},
					Annotations: map[string]string{
						annotationName:                    expectedAnnotationValue,
						extsecrets.SchemaObjectAnnotation: expectedSchemaAnnotation,
					},
				},
			},
		},
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

	testhelpers.AssertLabel(t, labelName, expetedLabelValue, secret.ObjectMeta, message)
	testhelpers.AssertAnnotation(t, annotationName, expectedAnnotationValue, secret.ObjectMeta, message)
	testhelpers.AssertAnnotation(t, extsecrets.SchemaObjectAnnotation, expectedSchemaAnnotation, secret.ObjectMeta, message)

	// replicated secrets
	secret, message = testhelpers.RequireSecretExists(t, kubeClient, "jx-staging", secretName)
	testhelpers.AssertSecretEntryEquals(t, secret, propertyName, expectedPropertyValue, message)
}
