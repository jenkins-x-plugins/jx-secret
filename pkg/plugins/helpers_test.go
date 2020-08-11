package plugins_test

import (
	"testing"

	"github.com/jenkins-x/jx-secret/pkg/plugins"
	"github.com/stretchr/testify/assert"
)

const (
	amd64 = "amd64"
)

func TestVaultPlugin(t *testing.T) {
	t.Parallel()

	v := plugins.VaultVersion
	plugin := plugins.CreateVaultPlugin(v)

	assert.Equal(t, plugins.VaultPluginName, plugin.Name, "plugin.Name")
	assert.Equal(t, plugins.VaultPluginName, plugin.Spec.Name, "plugin.Spec.Name")

	foundLinux := false
	foundWindows := false
	for _, b := range plugin.Spec.Binaries {
		switch b.Goarch {
		case amd64:
			switch b.Goos {
			case "Linux":
				foundLinux = true
				assert.Equal(t, "https://releases.hashicorp.com/vault/"+v+"/vault_"+v+"_linux_amd64.zip", b.URL, "URL for linux binary")
				t.Logf("found linux binary URL %s", b.URL)
			case "Windows":
				foundWindows = true
				assert.Equal(t, "https://releases.hashicorp.com/vault/"+v+"/vault_"+v+"_windows_amd64.zip", b.URL, "URL for windows binary")
				t.Logf("found windows binary URL %s", b.URL)
			case "Darwin":
				foundWindows = true
				assert.Equal(t, "https://releases.hashicorp.com/vault/"+v+"/vault_"+v+"_darwin_amd64.zip", b.URL, "URL for windows binary")
				t.Logf("found windows binary URL %s", b.URL)
			}
		}
	}
	assert.True(t, foundLinux, "did not find a linux binary in the plugin %#v", plugin)
	assert.True(t, foundWindows, "did not find a windows binary in the plugin %#v", plugin)
}
