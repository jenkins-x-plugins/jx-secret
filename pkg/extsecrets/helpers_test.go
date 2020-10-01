package extsecrets

import (
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx-helpers/v3/pkg/yamls"
	v1 "github.com/jenkins-x/jx-secret/pkg/apis/external/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalSuccess(t *testing.T) {
	fileName := filepath.Join("test_data", "lighthouse-oauth-token.yaml")
	assert.FileExists(t, fileName)
	es := &v1.ExternalSecret{}
	err := yamls.LoadFile(fileName, es)
	require.NoError(t, err, "failed to load file %s", fileName)

	t.Logf("loaded %#v", es)

	assert.Equal(t, "lighthouse-oauth-token", es.Name, "es.Name")

	assert.Equal(t, "vault", es.Spec.BackendType, "es.SchemaSpec.BackendType")
	assert.Equal(t, "kubernetes", es.Spec.VaultMountPoint, "es.SchemaSpec.VaultMountPoint")
	assert.Equal(t, "vault-infra", es.Spec.VaultRole, "es.SchemaSpec.VaultRole")

	require.NotNil(t, es.Spec.Template, "es.SchemaSpec.Template")
	require.NotNil(t, es.Spec.Template.Metadata, "es.SchemaSpec.Template.Metadata")
	assert.Len(t, es.Spec.Template.Metadata.Labels, 4, "es.SchemaSpec.Template.Metadata.Labels")
	assert.Equal(t, "Opaque", es.Spec.Template.Type, "es.SchemaSpec.Template.Type")

	require.Len(t, es.Spec.Data, 1, "es.SchemaSpec.Data")

	d1 := es.Spec.Data[0]
	assert.Equal(t, "oauth", d1.Name, "es.SchemaSpec.Data[0].Name")
	assert.Equal(t, "secret/data/jx/pipelineUser", d1.Key, "es.SchemaSpec.Data[0].Key")
	assert.Equal(t, "token", d1.Property, "es.SchemaSpec.Data[0].Property")

	require.NotNil(t, es.Status, "es.Status")
	assert.Equal(t, "SUCCESS", es.Status.Status, "es.Status.Status")
}

func TestUnmarshalFailure(t *testing.T) {
	fileName := filepath.Join("test_data", "knative-docker-user-pass.yaml")
	assert.FileExists(t, fileName)
	es := &v1.ExternalSecret{}
	err := yamls.LoadFile(fileName, es)
	require.NoError(t, err, "failed to load file %s", fileName)

	t.Logf("loaded %#v", es)

	assert.Equal(t, "knative-docker-user-pass", es.Name, "es.Name")

	assert.Equal(t, "vault", es.Spec.BackendType, "es.SchemaSpec.BackendType")
	assert.Equal(t, "kubernetes", es.Spec.VaultMountPoint, "es.SchemaSpec.VaultMountPoint")
	assert.Equal(t, "vault-infra", es.Spec.VaultRole, "es.SchemaSpec.VaultRole")

	require.NotNil(t, es.Spec.Template, "es.SchemaSpec.Template")
	require.NotNil(t, es.Spec.Template.Metadata, "es.SchemaSpec.Template.Metadata")
	assert.Len(t, es.Spec.Template.Metadata.Annotations, 1, "es.SchemaSpec.Template.Metadata.Annotations")
	assert.Equal(t, "kubernetes.io/basic-auth", es.Spec.Template.Type, "es.SchemaSpec.Template.Type")

	require.Len(t, es.Spec.Data, 2, "es.SchemaSpec.Data")

	d1 := es.Spec.Data[0]
	assert.Equal(t, "password", d1.Name, "es.SchemaSpec.Data[0].Name")
	assert.Equal(t, "secret/data/knative/docker/user/pass", d1.Key, "es.SchemaSpec.Data[0].Key")
	assert.Equal(t, "password", d1.Property, "es.SchemaSpec.Data[0].Property")

	require.NotNil(t, es.Status, "es.Status")
	assert.Equal(t, "ERROR, Status 404", es.Status.Status, "es.Status.Status")

}
