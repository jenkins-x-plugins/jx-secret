package populate_test

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	jxcore "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"
	"github.com/jenkins-x/jx-helpers/v3/pkg/maps"

	"github.com/jenkins-x-plugins/jx-secret/pkg/cmd/populate"
	"github.com/jenkins-x-plugins/jx-secret/pkg/cmd/populate/templatertesting"
	"github.com/jenkins-x-plugins/jx-secret/pkg/extsecrets"
	"github.com/jenkins-x-plugins/jx-secret/pkg/extsecrets/testsecrets"
	"github.com/jenkins-x-plugins/jx-secret/pkg/schemas"
	"github.com/jenkins-x-plugins/secretfacade/pkg/secretstore"
	secretstorefake "github.com/jenkins-x-plugins/secretfacade/testing/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/fake"
)

func runPopulateTestCases(t *testing.T, storeType secretstore.SecretStoreType, folder string, secretLocation string, mavenSecretName string, nexusSecretName string, extSecrets map[string]*secretstore.SecretValue, useSecretNameForKey bool, assertionFunc func(t *testing.T, fakeStore *secretstorefake.FakeSecretStore, mavenSettings string)) {

	ns := "jx"
	expectedMavenSettingsFile := filepath.Join("test_data", "populate", "expected", "jenkins-maven-settings", "settings.xml", "nexus.xml")
	require.FileExists(t, expectedMavenSettingsFile)
	expectedMaveSettingsData, err := ioutil.ReadFile(expectedMavenSettingsFile)
	require.NoError(t, err, "failed to load file %s", expectedMavenSettingsFile)

	schemaFile := filepath.Join("test_data", "populate", "secret-schema.yaml")
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
	}

	_, o := populate.NewCmdPopulate()
	o.Dir = fmt.Sprintf("test_data/populate/%s", folder)
	o.NoWait = true
	o.Namespace = ns
	o.BootSecretNamespace = ns
	o.KubeClient = fake.NewSimpleClientset(testsecrets.AddVaultSecrets(kubeObjects...)...)
	fakeFactory := secretstorefake.FakeSecretManagerFactory{}
	o.SecretStoreManagerFactory = &fakeFactory
	_, err = fakeFactory.NewSecretManager(storeType)
	assert.NoError(t, err)
	fakeStore := fakeFactory.GetSecretStore()
	for k, v := range extSecrets {
		err = fakeStore.SetSecret(secretLocation, k, v)
		assert.NoError(t, err)
	}

	dynObjects := testsecrets.LoadExtSecretDir(t, ns, filepath.Join("test_data", "populate", folder))
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

	assertionFunc(t, fakeStore, string(expectedMaveSettingsData))

	// Store Maven secret so we can detect diff after running populate a second time
	firstMavenSettingsSecret, err := fakeStore.GetSecret(secretLocation, mavenSecretName, "settingsXml")
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

			var secretValue string
			if useSecretNameForKey {
				secretValue, _ = fakeStore.GetSecret(secretLocation, es.Name, d.Property)
			} else {
				secretValue, _ = fakeStore.GetSecret(secretLocation, d.Key, d.Property)
			}
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
	o.Dir = fmt.Sprintf("test_data/populate/%s", folder)
	o.NoWait = true
	o.Namespace = ns
	o.KubeClient = fake.NewSimpleClientset(testsecrets.AddVaultSecrets(kubeObjects...)...)
	o.SecretStoreManagerFactory = &fakeFactory

	dynObjects = testsecrets.LoadExtSecretDir(t, ns, filepath.Join("test_data", "populate", folder))
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

	err = fakeStore.SetSecret(secretLocation, nexusSecretName, &secretstore.SecretValue{
		PropertyValues: map[string]string{
			"password": "my-new-nexus-password",
		}})
	assert.NoError(t, err)
	err = o.Run()
	require.NoError(t, err, "failed to invoke Run()")

	// Assert re-retrieve Maven settings secret has been modified due to presence of new secrets
	secondMavenSettingsSecret, err := fakeStore.GetSecret(secretLocation, mavenSecretName, "settingsXml")
	assert.NoError(t, err)
	assert.NotEmpty(t, secondMavenSettingsSecret)
	assert.NotEqual(t, firstMavenSettingsSecret, secondMavenSettingsSecret)

}

func TestPopulate(t *testing.T) {
	type testCase struct {
		backendTypePath     string
		secretLocation      string
		mavenSecretName     string
		nexusSecretName     string
		extSecrets          map[string]*secretstore.SecretValue
		useSecretNameForKey bool
		assertionFunc       func(t *testing.T, fakeStore *secretstorefake.FakeSecretStore, mavenSettings string)
	}
	gcpLocation := "123456"
	vaultLocation := "https://127.0.0.1:8200"
	azureLocation := "azureSuperSecretVault"
	kubeLocation := "jx"
	for _, folder := range []testCase{
		{"vaultsecrets",
			vaultLocation,
			"secret/data/jx/mavenSettings",
			"secret/data/nexus",
			map[string]*secretstore.SecretValue{
				"secret/data/sonatype": {
					PropertyValues: map[string]string{
						"username": "my-sonatype-username",
						"password": "my-sonatype-password",
					},
				},
				"secret/data/gpg": {
					PropertyValues: map[string]string{
						"passphrase": "my-secret-gpg-passphrase",
					},
				},
			},
			false,
			func(t *testing.T, fakeStore *secretstorefake.FakeSecretStore, mavenSettings string) {
				fakeStore.AssertValueEquals(t, vaultLocation, "secret/data/jx/adminUser", "username", "admin")
				fakeStore.AssertHasValue(t, vaultLocation, "secret/data/jx/adminUser", "password")
				fakeStore.AssertHasValue(t, vaultLocation, "secret/data/lighthouse/hmac", "hmac")
				fakeStore.AssertValueEquals(t, vaultLocation, "secret/data/jx/pipelineUser", "token", "gitoperatorpassword")
				fakeStore.AssertHasValue(t, vaultLocation, "secret/data/knative/docker/user/pass", "password")
				fakeStore.AssertValueEquals(t, vaultLocation, "secret/data/jx/mavenSettings", "settingsXml", mavenSettings)

			}},
		{"gsmsecrets",
			gcpLocation,
			"jx-maven-settings",
			"nexus",
			map[string]*secretstore.SecretValue{
				"sonatype": {
					PropertyValues: map[string]string{
						"username": "my-sonatype-username",
						"password": "my-sonatype-password",
					},
				},
				"gpg": {
					PropertyValues: map[string]string{
						"passphrase": "my-secret-gpg-passphrase",
					},
				},
			},
			false,
			func(t *testing.T, fakeStore *secretstorefake.FakeSecretStore, mavenSettings string) {
				fakeStore.AssertValueEquals(t, gcpLocation, "jx-admin-user", "username", "admin")
				fakeStore.AssertHasValue(t, gcpLocation, "jx-admin-user", "password")
				fakeStore.AssertHasValue(t, gcpLocation, "lighthouse-hmac", "")
				fakeStore.AssertValueEquals(t, gcpLocation, "jx-pipeline-user", "token", "gitoperatorpassword")
				fakeStore.AssertHasValue(t, gcpLocation, "knative-docker-user-pass", "password")
				fakeStore.AssertValueEquals(t, gcpLocation, "jx-maven-settings", "settingsXml", mavenSettings)

			}},
		{"azuresecrets",
			azureLocation,
			"jx-maven-settings",
			"nexus",
			map[string]*secretstore.SecretValue{
				"sonatype": {
					PropertyValues: map[string]string{
						"username": "my-sonatype-username",
						"password": "my-sonatype-password",
					},
				},
				"gpg": {
					PropertyValues: map[string]string{
						"passphrase": "my-secret-gpg-passphrase",
					},
				},
			},
			false,
			func(t *testing.T, fakeStore *secretstorefake.FakeSecretStore, mavenSettings string) {
				fakeStore.AssertValueEquals(t, azureLocation, "jx-admin-user", "username", "admin")
				fakeStore.AssertHasValue(t, azureLocation, "jx-admin-user", "password")
				fakeStore.AssertHasValue(t, azureLocation, "lighthouse-hmac", "")
				fakeStore.AssertValueEquals(t, azureLocation, "jx-pipeline-user", "token", "gitoperatorpassword")
				fakeStore.AssertHasValue(t, azureLocation, "knative-docker-user-pass", "password")
				fakeStore.AssertValueEquals(t, azureLocation, "jx-maven-settings", "settingsXml", mavenSettings)
			}},
		{"kubesecrets",
			kubeLocation,
			"jenkins-maven-settings",
			"nexus",
			map[string]*secretstore.SecretValue{
				"sonatype": {
					PropertyValues: map[string]string{
						"username": "my-sonatype-username",
						"password": "my-sonatype-password",
					},
				},
				"gpg": {
					PropertyValues: map[string]string{
						"passphrase": "my-secret-gpg-passphrase",
					},
				},
			},
			true,
			func(t *testing.T, fakeStore *secretstorefake.FakeSecretStore, mavenSettings string) {
				fakeStore.AssertValueEquals(t, kubeLocation, "jenkins-x-bucketrepo", "username", "admin")
				fakeStore.AssertHasValue(t, kubeLocation, "jenkins-x-bucketrepo", "password")
				fakeStore.AssertHasValue(t, kubeLocation, "lighthouse-hmac-token", "hmac")
				fakeStore.AssertValueEquals(t, kubeLocation, "lighthouse-oauth-token", "token", "gitoperatorpassword")
				fakeStore.AssertHasValue(t, kubeLocation, "knative-docker-user-pass", "password")
				fakeStore.AssertValueEquals(t, kubeLocation, "jenkins-maven-settings", "settingsXml", mavenSettings)
			}},
	} {
		runPopulateTestCases(t, secretstore.SecretStoreTypeVault, folder.backendTypePath, folder.secretLocation, folder.mavenSecretName, folder.nexusSecretName, folder.extSecrets, folder.useSecretNameForKey, folder.assertionFunc)
	}
}

func TestPopulateFromFileSystem(t *testing.T) {
	ns := "jx"
	vaultLocation := "https://127.0.0.1:8200"
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
	o.Dir = "test_data/populate_filesystem"
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
	secret, err := secretStore.GetSecret(vaultLocation, "secret/data/jx/pipelineUser", "token")
	secretStore.AssertHasValue(t, vaultLocation, "secret/data/jx/pipelineUser", "token")
	secretStore.AssertValueEquals(t, vaultLocation, "secret/data/jx/pipelineUser", "token", "gitoperatorpassword")
	assert.Equal(t, "gitoperatorpassword", secret)
}

func TestPopulateFromHelmSecrets(t *testing.T) {
	ns := "jx"
	secretLocation := "azureSuperSecretVault"
	//secretLocation := "jx"

	_, o := populate.NewCmdPopulate()
	o.Dir = "test_data/populate_helm_secrets"
	o.NoWait = true
	o.Namespace = ns
	o.BootSecretNamespace = ns
	fakeFactory := secretstorefake.FakeSecretManagerFactory{}
	o.SecretStoreManagerFactory = &fakeFactory
	o.KubeClient = fake.NewSimpleClientset()

	extSecretsDir := filepath.Join(o.Dir, "extsecrets")
	dynObjects := testsecrets.LoadExtSecretDir(t, ns, extSecretsDir)
	scheme := runtime.NewScheme()

	o.HelmSecretFolder = filepath.Join(o.Dir, "fake-helm-secrets")
	require.NotEmpty(t, dynObjects, "failed to load ExternaSecrets from dir %s", extSecretsDir)
	fakeDynClient := testsecrets.NewFakeDynClient(scheme, dynObjects...)

	var err error
	o.SecretClient, err = extsecrets.NewClient(fakeDynClient)
	require.NoError(t, err, "failed to create secret client")

	err = o.Run()
	require.NoError(t, err, "failed to invoke Run()")

	fakeStore := fakeFactory.GetSecretStore()
	fakeStore.AssertValueEquals(t, secretLocation, "lighthouse-oauth-token", "token", "fake-secret-value")
	fakeStore.AssertValueEquals(t, secretLocation, "secret-with-stringdata", "token", "token-value")
}

func TestParseRequirements(t *testing.T) {
	dir := filepath.Join("test_data", "load_requirements")
	requirementsResource, _, err := jxcore.LoadRequirementsConfig(dir, false)
	require.NoError(t, err, "Failed to load requirements from dir %s", dir)
	req := &requirementsResource.Spec
	expectedRegistry := "123456789012.dkr.ecr.ap-southeast-2.amazonaws.com"
	assert.Equal(t, expectedRegistry, req.Cluster.Registry, "cluster.registry")

	requirementsMap, err := populate.CreateRequirementsMap(req)
	require.NoError(t, err, "failed to create requirements map")

	value := maps.GetMapValueAsStringViaPath(requirementsMap, "cluster.registry")
	assert.Equal(t, expectedRegistry, value, "cluster.registry on the requirementsMap")
}
