package edit_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/jenkins-x-plugins/jx-secret/pkg/cmd/edit"
	"github.com/jenkins-x-plugins/jx-secret/pkg/extsecrets"
	"github.com/jenkins-x-plugins/jx-secret/pkg/extsecrets/testsecrets"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner/fakerunner"
	fakeinput "github.com/jenkins-x/jx-helpers/v3/pkg/input/fake"
	"github.com/jenkins-x/jx-helpers/v3/pkg/testhelpers"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func TestEditLocal(t *testing.T) {
	_, o := edit.NewCmdEdit()
	scheme := runtime.NewScheme()

	ns := "jx"

	kubeObjects := []runtime.Object{
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "knative-docker-user-pass",
				Namespace: ns,
			},
			Data: map[string][]byte{
				"username": []byte("dummyValue"),
			},
		},
	}

	o.Namespace = ns
	o.KubeClient = fake.NewSimpleClientset(kubeObjects...)

	var err error
	dynObjects := testsecrets.LoadExtSecretDir(t, ns, filepath.Join("test_data", "local"))
	fakeDynClient := testsecrets.NewFakeDynClient(scheme, dynObjects...)
	o.SecretClient, err = extsecrets.NewClient(fakeDynClient)
	require.NoError(t, err, "failed to create fake extsecrets Client")

	runner := &fakerunner.FakeRunner{}
	o.CommandRunner = runner.Run

	/* #nosec */
	const expectedDockerPwd = "dummyDockerPwd" //NOSONAR
	const expectedPipelineToken = "dummyPipelineToken"
	input := &fakeinput.FakeInput{
		Values: map[string]string{
			"knative-docker-user-pass.password": expectedDockerPwd,
			"lighthouse-oauth-token.token":      expectedPipelineToken,
		},
	}
	o.Input = input

	err = o.Run()
	require.NoError(t, err, "failed to run edit")

	ctx := context.TODO()
	list, err := o.KubeClient.CoreV1().Secrets(ns).List(ctx, metav1.ListOptions{})
	require.NoError(t, err, "failed to list secrets in namespace %s", ns)
	for _, s := range list.Items {
		t.Logf("found secret %s with data %#v\n", s.Name, s.Data)
	}
	require.NotEmpty(t, list.Items, "no Secrets found in namespace %s", ns)

	secret, message := testhelpers.RequireSecretExists(t, o.KubeClient, ns, "knative-docker-user-pass")
	testhelpers.AssertSecretEntryEquals(t, secret, "password", expectedDockerPwd, message)

	secret, message = testhelpers.RequireSecretExists(t, o.KubeClient, ns, "lighthouse-oauth-token")
	testhelpers.AssertSecretEntryEquals(t, secret, "token", expectedPipelineToken, message)
}
