package testsecrets

import (
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jenkins-x/jx-helpers/pkg/yamls"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// LoadExtSecretFiles loads the given YAML files as external secrets for a test case
func LoadExtSecretFiles(t *testing.T, ns string, fileNames ...string) []runtime.Object {
	var dynObjects []runtime.Object
	for _, path := range fileNames {
		require.FileExists(t, path)

		u := &unstructured.Unstructured{}
		err := yamls.LoadFile(path, u)
		require.NoError(t, err, "failed to load file %s", path)
		u.SetNamespace(ns)
		dynObjects = append(dynObjects, u)
	}
	return dynObjects
}

// LoadExtSecretDir loads the given YAML files in the given directory as external secrets for a test case
func LoadExtSecretDir(t *testing.T, ns, dir string) []runtime.Object {
	files, err := ioutil.ReadDir(dir)
	require.NoError(t, err, "failed to read dir %s", dir)
	var extSecrets []string
	for _, f := range files {
		name := f.Name()
		if !f.IsDir() && strings.HasSuffix(name, ".yaml") {
			extSecrets = append(extSecrets, filepath.Join(dir, name))
		}
	}
	return LoadExtSecretFiles(t, ns, extSecrets...)
}
