package edit_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jenkins-x-plugins/jx-secret/pkg/cmd/convert/edit"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/stretchr/testify/assert"

	"github.com/jenkins-x-plugins/jx-secret/pkg/apis/mapping/v1alpha1"
	"github.com/jenkins-x-plugins/jx-secret/pkg/secretmapping"
	"github.com/stretchr/testify/require"
)

func TestCmdSecretsMappingEdit(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		callback func(t *testing.T, sm *v1alpha1.SecretMapping)
		wantErr  bool
		fail     bool
	}{
		{
			name: "gsm_defaults_add",
			callback: func(t *testing.T, sm *v1alpha1.SecretMapping) {
				assert.Equal(t, 2, len(sm.Spec.Secrets), "should have found 2 mappings")
				for _, secret := range sm.Spec.Secrets {
					assert.Equal(t, "foo", secret.GcpSecretsManager.ProjectID, "secret.GcpSecretsManager.ProjectID")
					assert.Equal(t, "bar", secret.GcpSecretsManager.UniquePrefix, "secret.GcpSecretsManager.UniquePrefix")
					assert.Equal(t, "latest", secret.GcpSecretsManager.Version, "secret.GcpSecretsManager.Version")
				}

				assert.Equal(t, "foo", sm.Spec.Defaults.GcpSecretsManager.ProjectID, "sm.Spec.Defaults.GcpSecretsManager.ProjectID")
				assert.Equal(t, "bar", sm.Spec.Defaults.GcpSecretsManager.UniquePrefix, "secret.GcpSecretsManager.UniquePrefix")
			},
		},
		{
			name: "gsm_defaults_dont_replace",
			callback: func(t *testing.T, sm *v1alpha1.SecretMapping) {
				assert.Equal(t, 2, len(sm.Spec.Secrets), "should have found 2 mappings")
				assert.Equal(t, "phill", sm.Spec.Secrets[0].GcpSecretsManager.ProjectID, "secret.GcpSecretsManager.ProjectID")
				assert.Equal(t, "collins", sm.Spec.Secrets[0].GcpSecretsManager.UniquePrefix, "secret.GcpSecretsManager.UniquePrefix")
				assert.Equal(t, "1", sm.Spec.Secrets[0].GcpSecretsManager.Version, "secret.GcpSecretsManager.Version")
				assert.Equal(t, "foo", sm.Spec.Secrets[1].GcpSecretsManager.ProjectID, "secret.GcpSecretsManager.ProjectID")
				assert.Equal(t, "latest", sm.Spec.Secrets[1].GcpSecretsManager.Version, "secret.GcpSecretsManager.Version")
			},
		},
		{
			name: "asm_defaults_add",
			callback: func(t *testing.T, sm *v1alpha1.SecretMapping) {
				assert.Equal(t, 2, len(sm.Spec.Secrets), "should have found 2 mappings")
				for _, secret := range sm.Spec.Secrets {
					assert.Equal(t, "us-east-2", secret.AwsSecretsManager.Region, "secret.AwsSecretsManager.Region")
				}

				assert.Equal(t, "us-east-2", sm.Spec.Defaults.AwsSecretsManager.Region, "sm.Spec.Defaults.AwsSecretsManager.Region")
			},
		},
	}
	tmpDir := t.TempDir()

	for i, tt := range tests {
		if tt.name == "" {
			tt.name = fmt.Sprintf("test%d", i)
		}
		t.Logf("running test %s", tt.name)
		dir := filepath.Join(tmpDir)

		err := os.MkdirAll(dir, files.DefaultDirWritePermissions)
		require.NoError(t, err, "failed to create dir %s", dir)

		localSecretsFile := filepath.Join("test_data", tt.name)
		err = files.CopyDir(localSecretsFile, dir, true)
		require.NoError(t, err, "failed to copy %s to %s", localSecretsFile, dir)
		cmd, _ := edit.NewCmdSecretMappingEdit()
		tt.args = append(tt.args, "--dir", dir)

		err = cmd.ParseFlags(tt.args)
		require.NoError(t, err, "failed to parse arguments %#v for test %s", tt.args, tt.name)

		old := os.Args
		os.Args = tt.args
		err = cmd.RunE(cmd, tt.args)
		if err != nil {
			if tt.fail {
				t.Logf("got exected failure for test %s: %s", tt.name, err.Error())
				continue
			}
			t.Errorf("test %s reported error: %s", tt.name, err)
			continue
		}
		os.Args = old

		secretMapping, _, err := secretmapping.LoadSecretMapping(dir, true)
		require.NoError(t, err, "failed to load requirements from dir %s", dir)

		if tt.callback != nil {
			tt.callback(t, secretMapping)
		}
	}
}
