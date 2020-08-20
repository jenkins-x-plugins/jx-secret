package populate_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner/fakerunner"
	"github.com/jenkins-x/jx-secret/pkg/cmd/populate"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets/testsecrets"
	"github.com/jenkins-x/jx-secret/pkg/plugins"
	"github.com/jenkins-x/jx-secret/pkg/schemas"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	dynfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
)

func TestPopulate(t *testing.T) {
	vaultBin, err := plugins.GetVaultBinary(plugins.VaultVersion)
	require.NoError(t, err, "failed to find vault binary plugin")

	expectedMavenSettingsFile := filepath.Join("test_data", "expected-maven-settings.xml")
	require.FileExists(t, expectedMavenSettingsFile)
	expectedMaveSettingsData, err := ioutil.ReadFile(expectedMavenSettingsFile)
	require.NoError(t, err, "failed to load file %s", expectedMavenSettingsFile)

	_, o := populate.NewCmdPopulate()
	o.Dir = "test_data"
	o.NoWait = true
	scheme := runtime.NewScheme()

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

	o.Namespace = ns
	o.KubeClient = fake.NewSimpleClientset(testsecrets.AddVaultSecrets(kubeObjects...)...)

	schemaFile := filepath.Join("test_data", "secret-schema.yaml")
	schema, err := schemas.LoadSchemaFile(schemaFile)

	require.NoError(t, err, "failed to load schema file %s")
	dynObjects := testsecrets.LoadExtSecretDir(t, ns, filepath.Join("test_data", "secrets"))

	for _, r := range dynObjects {
		u, ok := r.(*unstructured.Unstructured)
		if ok && u != nil {
			name := u.GetName()
			obj := schema.Spec.FindObject(name)
			if obj != nil {
				ann := u.GetAnnotations()
				if ann == nil {
					ann = map[string]string{}
				}
				value, err := schemas.ToAnnotationString(obj)
				require.NoError(t, err, "failed to create annotation value for schema %#v on secret %s", obj, name)
				ann[extsecrets.SchemaObjectAnnotation] = value
				u.SetAnnotations(ann)
			}
		}
	}

	fakeDynClient := dynfake.NewSimpleDynamicClient(scheme, dynObjects...)
	o.SecretClient, err = extsecrets.NewClient(fakeDynClient)
	require.NoError(t, err, "failed to create fake extsecrets Client")

	runner := &fakerunner.FakeRunner{}
	o.CommandRunner = runner.Run

	err = o.Run()
	require.NoError(t, err, "failed to run edit")

	secretMaps := testsecrets.LoadFakeVaultSecrets(t, runner.OrderedCommands, vaultBin)

	secretMaps.AssertHasValue(t, "secret/jx/adminUser", "username")
	secretMaps.AssertHasValue(t, "secret/jx/adminUser", "password")
	secretMaps.AssertHasValue(t, "secret/lighthouse/hmac", "token")
	secretMaps.AssertHasValue(t, "secret/jx/pipelineUser", "token")
	secretMaps.AssertHasValue(t, "secret/knative/docker/user/pass", "password")

	secretMaps.AssertValueEquals(t, "secret/jx/mavenSettings", "settingsXml", string(expectedMaveSettingsData))
}
