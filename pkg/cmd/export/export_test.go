package export_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx-helpers/v3/pkg/testhelpers"
	"github.com/jenkins-x/jx-secret/pkg/cmd/export"
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

func TestExport(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "failed to create temp dir")

	_, o := export.NewCmdExport()
	scheme := runtime.NewScheme()

	ns := "jx"

	kubeObjects := []runtime.Object{
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "knative-docker-user-pass",
				Namespace: ns,
			},
			Data: map[string][]byte{
				"username": []byte("dummyDockerUsername"),
				"password": []byte("dummyDockerPassword"),
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "lighthouse-oauth-token",
				Namespace: ns,
			},
			Data: map[string][]byte{
				"oauth": []byte("dummyPipelineUserToken"),
			},
		},
	}
	o.Namespace = ns
	o.KubeClient = fake.NewSimpleClientset(kubeObjects...)

	dynObjects := testsecrets.LoadExtSecretDir(t, ns, filepath.Join("test_data", "secrets"))
	gvrToListKind := map[schema.GroupVersionResource]string{
		{Group: "kubernetes-client.io", Version: "v1", Resource: "externalsecrets"}: "ExternalSecretList",
	}
	fakeDynClient := dynfake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrToListKind, dynObjects...)
	o.SecretClient, err = extsecrets.NewClient(fakeDynClient)
	require.NoError(t, err, "failed to create fake extsecrets Client")

	fileName := filepath.Join(tmpDir, "secrets.yaml")
	o.OutFile = fileName

	err = o.Run()
	require.NoError(t, err, "failed to run export")

	assert.FileExists(t, fileName, "no file name generated")

	testhelpers.AssertYamlFilesEqual(t, filepath.Join("test_data", "expected.yaml"), fileName, "generated YAML")

}
