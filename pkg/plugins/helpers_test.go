package plugins_test

import (
	"testing"

	"github.com/jenkins-x/jx-secret/pkg/plugins"
	"github.com/stretchr/testify/assert"
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
		if b.Goos == "Linux" && b.Goarch == "amd64" {
			foundLinux = true
			assert.Equal(t, "https://releases.hashicorp.com/vault/"+v+"/vault_"+v+"_linux_amd64.zip", b.URL, "URL for linux binary")
			t.Logf("found linux binary URL %s", b.URL)
		} else if b.Goos == "Windows" && b.Goarch == "amd64" {
			foundWindows = true
			assert.Equal(t, "https://releases.hashicorp.com/vault/"+v+"/vault_"+v+"_windows_amd64.zip", b.URL, "URL for windows binary")
			t.Logf("found windows binary URL %s", b.URL)
		} else if b.Goos == "Darwin" && b.Goarch == "amd64" {
			foundWindows = true
			assert.Equal(t, "https://releases.hashicorp.com/vault/"+v+"/vault_"+v+"_darwin_amd64.zip", b.URL, "URL for windows binary")
			t.Logf("found windows binary URL %s", b.URL)
		}
	}
	assert.True(t, foundLinux, "did not find a linux binary in the plugin %#v", plugin)
	assert.True(t, foundWindows, "did not find a windows binary in the plugin %#v", plugin)
}
