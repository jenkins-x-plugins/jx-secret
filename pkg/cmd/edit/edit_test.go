package edit_test

import (
	"testing"

	"github.com/jenkins-x/jx-extsecret/pkg/cmd/edit"
	"github.com/jenkins-x/jx-extsecret/pkg/extsecrets"
	"github.com/jenkins-x/jx-extsecret/pkg/extsecrets/testsecrets"
	fakeinput "github.com/jenkins-x/jx-extsecret/pkg/input/fake"
	"github.com/jenkins-x/jx-extsecret/pkg/testhelpers"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	dynfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
)

func TestEdit(t *testing.T) {
	var err error
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
	dynObjects := testsecrets.LoadExtSecretFiles(t, ns, "knative-docker-user-pass.yaml", "lighthouse-oauth-token.yaml")

	fakeDynClient := dynfake.NewSimpleDynamicClient(scheme, dynObjects...)
	o.SecretClient, err = extsecrets.NewClient(fakeDynClient)
	require.NoError(t, err, "failed to create fake extsecrets Client")

	o.KubeClient = fake.NewSimpleClientset(kubeObjects...)

	runner := &testhelpers.FakeRunner{}
	o.CommandRunner = runner.Run

	input := &fakeinput.FakeInput{
		Values: map[string]string{
			"secret/data/knative/docker/user/pass.password": "dummyDockerPwd",
			"secret/data/jx/pipelineUser.token":             "dummyPipelineToken",
		},
	}
	o.Input = input

	err = o.Run()
	require.NoError(t, err, "failed to run edit")

	runner.ExpectResults(t,
		testhelpers.FakeResult{
			CLI: "vault kv put secret/jx/pipelineUser token=dummyPipelineToken",
		},
		testhelpers.FakeResult{
			CLI: "vault kv put secret/knative/docker/user/pass password=dummyDockerPwd",
		},
	)
}
