package importcmd_test

import (
	"path/filepath"
	"testing"

	"github.com/alecthomas/assert"
	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner/fakerunner"
	importcmd "github.com/jenkins-x/jx-secret/pkg/cmd/import"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets/testsecrets"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	dynfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
)

func TestImport(t *testing.T) {
	var err error
	_, o := importcmd.NewCmdImport()
	scheme := runtime.NewScheme()

	ns := "jx"
	dynObjects := testsecrets.LoadExtSecretFiles(t, ns, "knative-docker-user-pass.yaml", "lighthouse-oauth-token.yaml")

	fakeDynClient := dynfake.NewSimpleDynamicClient(scheme, dynObjects...)
	o.SecretClient, err = extsecrets.NewClient(fakeDynClient)
	o.KubeClient = fake.NewSimpleClientset(testsecrets.AddVaultSecrets()...)

	require.NoError(t, err, "failed to create fake extsecrets Client")

	runner := &fakerunner.FakeRunner{}
	o.CommandRunner = runner.Run

	o.File = filepath.Join("test_data", "import-file.yaml")

	err = o.Run()
	require.NoError(t, err, "failed to run import")

	runner.ExpectResults(t,
		fakerunner.FakeResult{
			CLI: "vault version",
		},
		fakerunner.FakeResult{
			CLI: "vault kv list secret",
		},
		fakerunner.FakeResult{
			CLI: "vault kv put secret/pipelineUser email=jenkins-x@googlegroups.com token=dummyPipelineUser username=jenkins-x-labs-bot",
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
