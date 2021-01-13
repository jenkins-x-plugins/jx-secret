package secretfacade_test

import (
	"testing"

	"github.com/alecthomas/assert"
	"github.com/jenkins-x/jx-helpers/v3/pkg/stringhelpers"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets/secretfacade"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets/testsecrets"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func TestVerifyFilter(t *testing.T) {
	scheme := runtime.NewScheme()

	var err error
	ns := "jx"

	kubeObjects := []runtime.Object{
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "jx-boot",
				Namespace: ns,
			},
			Data: map[string][]byte{
				"username": []byte("gitoperatorUsername"),
				"password": []byte("gitoperatorpassword"),
			},
		},
	}

	o := &secretfacade.Options{
		Dir:             "",
		Namespace:       "",
		SecretClient:    nil,
		KubeClient:      nil,
		ExternalSecrets: nil,
	}

	o.Dir = "test_data"
	o.Namespace = ns
	o.KubeClient = fake.NewSimpleClientset(kubeObjects...)

	dynObjects := testsecrets.LoadExtSecretDir(t, ns, "test_data")
	fakeDynClient := testsecrets.NewFakeDynClient(scheme, dynObjects...)

	o.SecretClient, err = extsecrets.NewClient(fakeDynClient)
	require.NoError(t, err, "failed to create fake extsecrets Client")

	secrets, err := o.VerifyAndFilter()
	require.NoError(t, err, "failed to run VerifyAndFilter")
	require.NotEmpty(t, secrets, "did not find any secrets")

	expectedFilteredSecrets := []string{"jx-pipeline-git-github-github"}
	for _, s := range secrets {
		object, err := s.SchemaObject()
		name := s.ExternalSecret.Name
		require.NoError(t, err, "failed to load schema object for secret %s", name)
		require.NotNil(t, object, "no schema object for secret %s", name)

		var propertyNames []string
		for _, p := range object.Properties {
			propertyNames = append(propertyNames, p.Name)
		}
		t.Logf("secret %s has schema properties %#v", name, propertyNames)

		if stringhelpers.StringArrayIndex(expectedFilteredSecrets, name) >= 0 {
			assert.Fail(t, "should have filtered out ExternalSecret", "for secret %s due to the same secret mapping locations", name)
		}
	}
}
