package importcmd_test

import (
	"path/filepath"
	"testing"

	importcmd "github.com/jenkins-x/jx-extsecret/pkg/cmd/import"
	"github.com/jenkins-x/jx-extsecret/pkg/extsecrets"
	"github.com/jenkins-x/jx-extsecret/pkg/extsecrets/testsecrets"
	"github.com/jenkins-x/jx-extsecret/pkg/testhelpers"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	dynfake "k8s.io/client-go/dynamic/fake"
)

func TestImport(t *testing.T) {
	var err error
	_, o := importcmd.NewCmdImport()
	scheme := runtime.NewScheme()

	ns := "jx"
	dynObjects := testsecrets.LoadExtSecretFiles(t, ns, "knative-docker-user-pass.yaml", "lighthouse-oauth-token.yaml")

	fakeDynClient := dynfake.NewSimpleDynamicClient(scheme, dynObjects...)
	o.SecretClient, err = extsecrets.NewClient(fakeDynClient)
	require.NoError(t, err, "failed to create fake extsecrets Client")

	runner := &testhelpers.FakeRunner{}
	o.CommandRunner = runner.Run

	o.File = filepath.Join("test_data", "import-file.yaml")

	err = o.Run()
	require.NoError(t, err, "failed to run import")

	runner.ExpectResults(t,
		testhelpers.FakeResult{
			CLI: "vault kv put secret/pipelineUser email=jenkins-x@googlegroups.com token=dummyPipelineUser username=jenkins-x-labs-bot",
		},
	)
}
