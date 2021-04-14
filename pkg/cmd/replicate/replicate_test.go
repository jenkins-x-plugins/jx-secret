package replicate_test

import (
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	v1 "github.com/jenkins-x-plugins/jx-secret/pkg/apis/external/v1"
	"github.com/jenkins-x-plugins/jx-secret/pkg/cmd/replicate"
	"github.com/jenkins-x-plugins/jx-secret/pkg/extsecrets"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/testhelpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yamls"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReplicateByName(t *testing.T) {
	callback := func(o *replicate.Options) {
		o.Name = []string{"knative-docker-user-pass", "lighthouse-oauth-token"}
	}

	AssertReplicate(t, callback)
}

func TestReplicateBySelector(t *testing.T) {
	callback := func(o *replicate.Options) {
		o.Selector = "secret.jenkins-x.io/replica-source=true"
	}

	AssertReplicate(t, callback)
}

func AssertReplicate(t *testing.T, callback func(o *replicate.Options)) {
	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "could not create temp dir")

	sourceData := filepath.Join("test_data")

	err = files.CopyDirOverwrite(sourceData, tmpDir)
	require.NoError(t, err, "failed to copy generated crds at %s to %s", sourceData, tmpDir)

	_, o := replicate.NewCmdReplicate()
	o.Dir = tmpDir

	callback(o)

	err = o.Run()
	require.NoError(t, err, "failed to replicate to external secrets in dir %s", tmpDir)

	t.Logf("replicated in test dir %s\n", tmpDir)

	require.Equal(t, []string{"jx-staging", "jx-production"}, o.To, "should have found the environment namespaces")

	for _, ns := range o.To {
		fileNames := []string{
			filepath.Join(o.NamespacesDir, ns, "lighthouse", "lighthouse-oauth-token.yaml"),
			filepath.Join(o.NamespacesDir, ns, "tekton", "knative-docker-user-pass.yaml"),
		}
		for _, file := range fileNames {
			if assert.FileExists(t, file, "should have generated file") {
				t.Logf("generated expected file %s", file)
			}

			es := &v1.ExternalSecret{}
			err = yamls.LoadFile(file, es)
			require.NoError(t, err, "failed to load file %s", file)

			testhelpers.AssertAnnotation(t, extsecrets.ReplicaAnnotation, "true", es.ObjectMeta, "replica tekton should be annotated")
			t.Logf("added annotation to tekton source file %s of %s: %s", file, extsecrets.ReplicaAnnotation, es.Annotations[extsecrets.ReplicaAnnotation])
		}

		assert.NoFileExists(t, filepath.Join(o.NamespacesDir, ns, "lighthouse", "lighthouse-hmac-token.yaml"), "should not have replicated the hmac token")
	}

	// lets verify we add a replication annotation to the source ExternalSecret to enable replication
	es := &v1.ExternalSecret{}
	tektonSourceFile := filepath.Join(o.NamespacesDir, "jx", "tekton", "knative-docker-user-pass.yaml")
	err = yamls.LoadFile(tektonSourceFile, es)
	require.NoError(t, err, "failed to load file %s", tektonSourceFile)

	testhelpers.AssertAnnotation(t, extsecrets.ReplicateToAnnotation, strings.Join(o.To, ","), es.ObjectMeta, "source tekton should be annotated")
	t.Logf("added annotation to tekton source file %s of %s: %s", tektonSourceFile, extsecrets.ReplicateToAnnotation, es.Annotations[extsecrets.ReplicateToAnnotation])
}
