package testsecrets

import (
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx-helpers/pkg/yamls"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// LoadExtSecretFiles loads the given YAML files as external secrets for a test case
func LoadExtSecretFiles(t *testing.T, ns string, fileNames ...string) []runtime.Object {
	var dynObjects []runtime.Object
	for _, f := range fileNames {
		path := filepath.Join("test_data", f)
		require.FileExists(t, path)

		u := &unstructured.Unstructured{}
		err := yamls.LoadFile(path, u)
		require.NoError(t, err, "failed to load file %s", path)
		u.SetNamespace(ns)
		dynObjects = append(dynObjects, u)
	}
	return dynObjects
}
