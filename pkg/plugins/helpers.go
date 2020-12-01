package plugins

import (
	"fmt"
	"strings"

	jenkinsv1 "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"
	"github.com/jenkins-x/jx-helpers/v3/pkg/extensions"
	"github.com/jenkins-x/jx-helpers/v3/pkg/homedir"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// VaultPluginName the default name of the vault plugin
	VaultPluginName = "vault"
)

// GetVaultBinary returns the path to the locally installed vault 3 extension
func GetVaultBinary(version string) (string, error) {
	if version == "" {
		version = VaultVersion
	}
	pluginBinDir, err := homedir.DefaultPluginBinDir()
	if err != nil {
		return "", err
	}
	plugin := CreateVaultPlugin(version)
	return extensions.EnsurePluginInstalled(plugin, pluginBinDir)
}

// CreateVaultPlugin creates the vault 3 plugin
func CreateVaultPlugin(version string) jenkinsv1.Plugin {
	binaries := extensions.CreateBinaries(func(p extensions.Platform) string {
		return fmt.Sprintf("https://releases.hashicorp.com/vault/%s/vault_%s_%s_%s.zip", version, version, strings.ToLower(p.Goos), strings.ToLower(p.Goarch))
	})

	plugin := jenkinsv1.Plugin{
		ObjectMeta: metav1.ObjectMeta{
			Name: VaultPluginName,
		},
		Spec: jenkinsv1.PluginSpec{
			SubCommand:  "vault",
			Binaries:    binaries,
			Description: "vault 3 binary",
			Name:        VaultPluginName,
			Version:     version,
		},
	}
	return plugin
}
