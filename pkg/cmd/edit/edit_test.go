package edit_test

import (
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner/fakerunner"
	fakeinput "github.com/jenkins-x/jx-helpers/v3/pkg/input/fake"
	"github.com/jenkins-x/jx-helpers/v3/pkg/testhelpers"
	"github.com/jenkins-x/jx-secret/pkg/cmd/edit"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets/testsecrets"
	"github.com/jenkins-x/jx-secret/pkg/plugins"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	dynfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
)

func TestEditVault(t *testing.T) {
	vaultBin, err := plugins.GetVaultBinary(plugins.VaultVersion)
	require.NoError(t, err, "failed to find vault binary plugin")

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
	o.KubeClient = fake.NewSimpleClientset(testsecrets.AddVaultSecrets(kubeObjects...)...)

	dynObjects := testsecrets.LoadExtSecretDir(t, ns, filepath.Join("test_data", "vault"))
	fakeDynClient := dynfake.NewSimpleDynamicClient(scheme, dynObjects...)
	o.SecretClient, err = extsecrets.NewClient(fakeDynClient)
	require.NoError(t, err, "failed to create fake extsecrets Client")

	runner := &fakerunner.FakeRunner{}
	o.CommandRunner = runner.Run

	input := &fakeinput.FakeInput{
		Values: map[string]string{
			"knative-docker-user-pass.password": "dummyDockerPwd",
			"lighthouse-oauth-token.token":      "dummyPipelineToken",
		},
	}
	o.Input = input

	err = o.Run()
	require.NoError(t, err, "failed to run edit")

	runner.ExpectResults(t,
		fakerunner.FakeResult{
			CLI: vaultBin + " version",
		},
		fakerunner.FakeResult{
			CLI: vaultBin + " kv list secret",
		},
		fakerunner.FakeResult{
			CLI: vaultBin + " kv put secret/jx/pipelineUser token=dummyPipelineToken",
		},
		fakerunner.FakeResult{
			CLI: vaultBin + " kv put secret/knative/docker/user/pass password=dummyDockerPwd", //NOSONAR
			Env: map[string]string{
				"VAULT_ADDR":  "https://127.0.0.1:8200",
				"VAULT_TOKEN": "dummyVaultToken",
			},
		},
	)

	// lets assert the vault env vars are setup correctly
	lastCommand := runner.OrderedCommands[len(runner.OrderedCommands)-1]
	vaultCaCert := lastCommand.Env["VAULT_CACERT"]
	assert.NotEmpty(t, vaultCaCert, "should have $VAULT_CACERT for command %s", cmdrunner.CLI(lastCommand))
	t.Logf("has $VAULT_CACERT %s\n", vaultCaCert)
}

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
	fakeDynClient := dynfake.NewSimpleDynamicClient(scheme, dynObjects...)
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

	secret, message := testhelpers.RequireSecretExists(t, o.KubeClient, ns, "knative-docker-user-pass")
	testhelpers.AssertSecretEntryEquals(t, secret, "password", expectedDockerPwd, message)

	secret, message = testhelpers.RequireSecretExists(t, o.KubeClient, ns, "lighthouse-oauth-token")
	testhelpers.AssertSecretEntryEquals(t, secret, "token", expectedPipelineToken, message)
}
