package replicate_test

import (
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jenkins-x/jx-helpers/pkg/files"
	"github.com/jenkins-x/jx-helpers/pkg/testhelpers"
	"github.com/jenkins-x/jx-helpers/pkg/yamls"
	v1 "github.com/jenkins-x/jx-secret/pkg/apis/external/v1"
	"github.com/jenkins-x/jx-secret/pkg/cmd/replicate"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReplicate(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "could not create temp dir")

	sourceData := filepath.Join("test_data")

	err = files.CopyDirOverwrite(sourceData, tmpDir)
	require.NoError(t, err, "failed to copy generated crds at %s to %s", sourceData, tmpDir)

	_, o := replicate.NewCmdReplicate()
	o.Dir = tmpDir
	o.Name = []string{"knative-docker-user-pass", "lighthouse-oauth-token"}

	err = o.Run()
	require.NoError(t, err, "failed to replicate to external secrets in dir %s", tmpDir)

	t.Logf("replicated in test dir %s\n", tmpDir)

	require.Equal(t, []string{"jx-staging", "jx-production"}, o.To, "should have found the environment namespaces")

	for _, ns := range o.To {
		files := []string{
			filepath.Join(o.NamespacesDir, ns, "lighthouse", "lighthouse-oauth-token.yaml"),
			filepath.Join(o.NamespacesDir, ns, "tekton", "knative-docker-user-pass.yaml"),
		}
		for _, file := range files {
			if assert.FileExists(t, file, "should have generated file") {
				t.Logf("generated expected file %s", file)
			}

			es := &v1.ExternalSecret{}
			err = yamls.LoadFile(file, es)
			require.NoError(t, err, "failed to load file %s", file)

			testhelpers.AssertAnnotation(t, extsecrets.ReplicaAnnotation, "true", es.ObjectMeta, "replica tekton should be annotated")
			t.Logf("added annotation to tekton source file %s of %s: %s", file, extsecrets.ReplicaAnnotation, es.Annotations[extsecrets.ReplicaAnnotation])
		}
	}

	// lets verify we add a replication annotation to the source ExternalSecret to enable replication
	es := &v1.ExternalSecret{}
	tektonSourceFile := filepath.Join(o.NamespacesDir, "jx", "tekton", "knative-docker-user-pass.yaml")
	err = yamls.LoadFile(tektonSourceFile, es)
	require.NoError(t, err, "failed to load file %s", tektonSourceFile)

	testhelpers.AssertAnnotation(t, extsecrets.ReplicateToAnnotation, strings.Join(o.To, ","), es.ObjectMeta, "source tekton should be annotated")
	t.Logf("added annotation to tekton source file %s of %s: %s", tektonSourceFile, extsecrets.ReplicateToAnnotation, es.Annotations[extsecrets.ReplicateToAnnotation])
}
