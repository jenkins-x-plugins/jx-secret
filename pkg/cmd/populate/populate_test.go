package populate_test

import (
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner/fakerunner"
	"github.com/jenkins-x/jx-secret/pkg/apis/schema/v1alpha1"
	"github.com/jenkins-x/jx-secret/pkg/cmd/populate"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets/testsecrets"
	"github.com/jenkins-x/jx-secret/pkg/plugins"
	"github.com/jenkins-x/jx-secret/pkg/schemas"
	"github.com/stretchr/testify/assert"
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

	schemaFile := filepath.Join("test_data", "secret-schema.yaml")
	schema, err := schemas.LoadSchemaFile(schemaFile)
	require.NoError(t, err, "failed to load schema file %s")

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

	dynObjects := testsecrets.LoadExtSecretDir(t, ns, filepath.Join("test_data", "secrets"))
	err = AddSchemaAnnotations(t, schema, dynObjects)
	require.NoError(t, err, "failed to add the schema annotations")

	fakeDynClient := dynfake.NewSimpleDynamicClient(scheme, dynObjects...)
	o.SecretClient, err = extsecrets.NewClient(fakeDynClient)
	require.NoError(t, err, "failed to create fake extsecrets Client")

	runner := &fakerunner.FakeRunner{}
	o.CommandRunner = runner.Run

	err = o.Run()
	require.NoError(t, err, "failed to invoke Run()")

	secretMaps := testsecrets.LoadFakeVaultSecrets(t, runner.OrderedCommands, vaultBin)

	secretMaps.AssertHasValue(t, "secret/jx/adminUser", "username")
	secretMaps.AssertHasValue(t, "secret/jx/adminUser", "password")
	secretMaps.AssertHasValue(t, "secret/lighthouse/hmac", "token")
	secretMaps.AssertHasValue(t, "secret/jx/pipelineUser", "token")
	secretMaps.AssertHasValue(t, "secret/knative/docker/user/pass", "password")

	secretMaps.AssertValueEquals(t, "secret/jx/mavenSettings", "settingsXml", string(expectedMaveSettingsData))

	esList, err := o.SecretClient.List(ns, metav1.ListOptions{})
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
			keyValues := secretMaps.Objects[d.Key]
			if keyValues == nil {
				// handle different key encodings for vault
				key2 := "secret" + strings.TrimPrefix(d.Key, "secret/data")
				keyValues = secretMaps.Objects[key2]
			}
			if keyValues != nil {
				value := keyValues[d.Name]
				if value == "" {
					value = keyValues[d.Property]
				}
				if value != "" {
					t.Logf("found value for ExternalSecret %s %s of %s", es.Name, d.Name, value)
					s.Data[d.Name] = []byte(value)
					s.Data[d.Property] = []byte(value)
				}
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

	dynObjects = testsecrets.LoadExtSecretDir(t, ns, filepath.Join("test_data", "secrets"))
	err = AddSchemaAnnotations(t, schema, dynObjects)
	require.NoError(t, err, "failed to add the schema annotations")

	fakeDynClient = dynfake.NewSimpleDynamicClient(scheme, dynObjects...)
	o.SecretClient, err = extsecrets.NewClient(fakeDynClient)
	require.NoError(t, err, "failed to create fake extsecrets Client")

	runner = &fakerunner.FakeRunner{}
	o.CommandRunner = runner.Run

	err = o.Run()
	require.NoError(t, err, "failed to invoke Run()")

	secretMaps2 := testsecrets.LoadFakeVaultSecrets(t, runner.OrderedCommands, vaultBin)

	// we should not have populated any more secrets now as all the default values are generated
	if secretMaps2.Objects != nil {
		for k, values := range secretMaps2.Objects {
			t.Logf("should not have populated secret %s with values %#v\n", k, values)
		}
	}

	assert.Empty(t, secretMaps2.Objects, "should not have populated any more secrets on the second run")
}

func AddSchemaAnnotations(t *testing.T, schema *v1alpha1.Schema, dynObjects []runtime.Object) error {
	var err error
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
				value := ""
				value, err = schemas.ToAnnotationString(obj)
				require.NoError(t, err, "failed to create annotation value for schema %#v on secret %s", obj, name)
				ann[extsecrets.SchemaObjectAnnotation] = value
				u.SetAnnotations(ann)
			}
		}
	}
	return err
}
