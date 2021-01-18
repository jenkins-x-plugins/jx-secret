package populate_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	secretstorefake "github.com/chrismellard/secretfacade/testing/fake"
	"github.com/jenkins-x/jx-secret/pkg/cmd/populate"
	"github.com/jenkins-x/jx-secret/pkg/cmd/populate/templatertesting"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets/testsecrets"
	"github.com/jenkins-x/jx-secret/pkg/schemas"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/fake"
)

func TestPopulate(t *testing.T) {

	ns := "jx"
	expectedMavenSettingsFile := filepath.Join("test_data", "expected", "jenkins-maven-settings", "settings.xml", "nexus.xml")
	require.FileExists(t, expectedMavenSettingsFile)
	expectedMaveSettingsData, err := ioutil.ReadFile(expectedMavenSettingsFile)
	require.NoError(t, err, "failed to load file %s", expectedMavenSettingsFile)

	schemaFile := filepath.Join("test_data", "secret-schema.yaml")
	schema, err := schemas.LoadSchemaFile(schemaFile)
	require.NoError(t, err, "failed to load schema file %s")

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

		// some other secrets used for templating the jenkins-maven-settings Secret
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nexus",
				Namespace: ns,
			},
			Data: map[string][]byte{
				"password": []byte("my-nexus-password"),
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sonatype",
				Namespace: ns,
			},
			Data: map[string][]byte{
				"username": []byte("my-sonatype-username"),
				"password": []byte("my-sonatype-password"),
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "gpg",
				Namespace: ns,
			},
			Data: map[string][]byte{
				"passphrase": []byte("my-secret-gpg-passphrase"),
			},
		},
	}

	_, o := populate.NewCmdPopulate()
	o.Dir = "test_data"
	o.NoWait = true
	o.Namespace = ns
	o.BootSecretNamespace = ns
	o.KubeClient = fake.NewSimpleClientset(testsecrets.AddVaultSecrets(kubeObjects...)...)
	fakeFactory := secretstorefake.FakeSecretManagerFactory{}
	o.SecretStoreManagerFactory = &fakeFactory

	dynObjects := testsecrets.LoadExtSecretDir(t, ns, filepath.Join("test_data", "secrets"))
	err = templatertesting.AddSchemaAnnotations(t, schema, dynObjects)
	require.NoError(t, err, "failed to add the schema annotations")

	scheme := runtime.NewScheme()
	fakeDynClient := testsecrets.NewFakeDynClient(scheme, dynObjects...)

	o.SecretClient, err = extsecrets.NewClient(fakeDynClient)
	require.NoError(t, err, "failed to create fake extsecrets Client")

	o.Backoff = &wait.Backoff{
		Steps:    5,
		Duration: 2 * time.Millisecond,
		Factor:   2.0,
		Jitter:   0.1,
	}

	err = o.Run()
	require.NoError(t, err, "failed to invoke Run()")

	fakeStore := fakeFactory.GetSecretStore()
	fakeStore.AssertValueEquals(t, "", "secret/data/jx/adminUser", "username", "admin")
	fakeStore.AssertHasValue(t, "", "secret/data/jx/adminUser", "password")
	fakeStore.AssertHasValue(t, "", "secret/data/lighthouse/hmac", "token")
	fakeStore.AssertValueEquals(t, "", "secret/data/jx/pipelineUser", "token", "gitoperatorpassword")
	fakeStore.AssertHasValue(t, "", "secret/data/knative/docker/user/pass", "password")
	fakeStore.AssertValueEquals(t, "", "secret/data/jx/mavenSettings", "settingsXml", string(expectedMaveSettingsData))

	// Store Maven secret so we can detect diff after running populate a second time
	firstMavenSettingsSecret, err := fakeStore.GetSecret("", "secret/data/jx/mavenSettings", "settingsXml")
	assert.NoError(t, err)
	assert.NotEmpty(t, firstMavenSettingsSecret)

	esList, err := o.SecretClient.List(ns)
	require.NoError(t, err, "failed to list the ExternalSecrets")

	for _, es := range esList {
		s := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      es.Name,
				Namespace: ns,
			},
			Data: map[string][]byte{},
		}

		for _, d := range es.Spec.Data {
			// Populate secret key value combination

			secretValue, _ := fakeStore.GetSecret("", d.Key, d.Property)
			if secretValue != "" {
				t.Logf("found value for ExternalSecret %s %s of %s", es.Name, d.Name, secretValue)
				s.Data[d.Property] = []byte(secretValue)
				s.Data[d.Name] = []byte(secretValue)

			}

		}
		if len(s.Data) > 0 {
			kubeObjects = append(kubeObjects, s)
		}
	}

	// lets rerun the populate and assert we have the same data
	_, o = populate.NewCmdPopulate()
	o.Dir = "test_data"
	o.NoWait = true
	o.Namespace = ns
	o.KubeClient = fake.NewSimpleClientset(testsecrets.AddVaultSecrets(kubeObjects...)...)
	o.SecretStoreManagerFactory = &fakeFactory

	dynObjects = testsecrets.LoadExtSecretDir(t, ns, filepath.Join("test_data", "secrets"))
	err = templatertesting.AddSchemaAnnotations(t, schema, dynObjects)
	require.NoError(t, err, "failed to add the schema annotations")
	fakeDynClient = testsecrets.NewFakeDynClient(scheme, dynObjects...)
	o.SecretClient, err = extsecrets.NewClient(fakeDynClient)
	require.NoError(t, err, "failed to create fake extsecrets Client")

	o.Backoff = &wait.Backoff{
		Steps:    5,
		Duration: 2 * time.Millisecond,
		Factor:   2.0,
		Jitter:   0.1,
	}

	err = o.Run()
	require.NoError(t, err, "failed to invoke Run()")

	// Assert re-retrieve Maven settings secret has been modified due to presence of new secrets
	secondMavenSettingsSecret, err := fakeStore.GetSecret("", "secret/data/jx/mavenSettings", "settingsXml")
	assert.NoError(t, err)
	assert.NotEmpty(t, secondMavenSettingsSecret)
	assert.NotEqual(t, firstMavenSettingsSecret, secondMavenSettingsSecret)

}

func TestPopulateFromFileSystem(t *testing.T) {
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

	_, o := populate.NewCmdPopulate()
	o.Dir = "test_data/filesystem"
	o.NoWait = true
	o.Namespace = ns
	o.BootSecretNamespace = ns
	o.Source = "filesystem"
	fakeFactory := secretstorefake.FakeSecretManagerFactory{}
	o.SecretStoreManagerFactory = &fakeFactory
	o.KubeClient = fake.NewSimpleClientset(testsecrets.AddVaultSecrets(kubeObjects...)...)

	err := o.Run()
	require.NoError(t, err, "failed to invoke Run()")

	secretStore := fakeFactory.GetSecretStore()
	secret, err := secretStore.GetSecret("", "secret/data/jx/pipelineUser", "token")
	secretStore.AssertHasValue(t, "", "secret/data/jx/pipelineUser", "token")
	secretStore.AssertValueEquals(t, "", "secret/data/jx/pipelineUser", "token", "gitoperatorpassword")
	assert.Equal(t, "gitoperatorpassword", secret)
}
