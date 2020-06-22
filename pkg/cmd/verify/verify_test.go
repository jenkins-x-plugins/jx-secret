package verify_test

import (
	"testing"

	"github.com/jenkins-x/jx-extsecret/pkg/cmd/verify"
	"github.com/jenkins-x/jx-extsecret/pkg/extsecrets"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"
)

func TestVerify(t *testing.T) {
	var err error
	_, o := verify.NewCmdVerify()
	scheme := runtime.NewScheme()
	fakeDynClient := fake.NewSimpleDynamicClient(scheme)
	o.Client, err = extsecrets.NewClient(fakeDynClient)
	require.NoError(t, err, "failed to create fake extsecrets Client")

	err = o.Run()
	require.NoError(t, err, "failed to run verify")
}
