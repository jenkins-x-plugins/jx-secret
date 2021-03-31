package edit

import (
	"fmt"
	"os"
	"path/filepath"

	jxcore "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"

	"github.com/jenkins-x-plugins/jx-secret/pkg/apis/mapping/v1alpha1"
	"github.com/jenkins-x-plugins/jx-secret/pkg/secretmapping"

	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x-plugins/jx-secret/pkg/rootcmd"

	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
)

// Options the CLI options for this command
type Options struct {
	Dir           string
	SecretMapping v1alpha1.SecretMapping
	Cmd           *cobra.Command
	Args          []string
	requirements  *jxcore.RequirementsConfig
}

var (
	cmdLong = templates.LongDesc(`
		Edits the local 'secret-mappings.yaml' file 
`)

	cmdExample = templates.Examples(`
		# edits the local 'secret-mappings.yaml' file 
		%s secretsmapping edit --gcp-project-id foo --cluster-name
`)
)

// NewCmdRequirementsEdit creates the new command
func NewCmdSecretMappingEdit() (*cobra.Command, *Options) {
	options := &Options{}
	cmd := &cobra.Command{
		Use:     "edit",
		Short:   "Edits the local 'secret-mappings.yaml' file",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			options.Cmd = cmd
			options.Args = args
			return options.Run()
		},
	}
	cmd.Flags().StringVarP(&options.Dir, "dir", "", "", "base directory containing '.jx/secret/mapping/secret-mappings.yaml' file")

	return cmd, options
}

// Run runs the command
func (o *Options) Run() error {
	var err error
	var requirementsResource *jxcore.Requirements
	requirementsResource, _, err = jxcore.LoadRequirementsConfig(o.Dir, false)
	if err != nil {
		return errors.Wrapf(err, "failed to load requirements in dir %s", o.Dir)
	}

	o.requirements = &requirementsResource.Spec

	if o.Dir == "" {
		o.Dir = filepath.Join(".jx", "gitops")
	}

	secretMapping, fileName, err := secretmapping.LoadSecretMapping(o.Dir, true)
	if err != nil {
		return err
	}
	if fileName == "" {
		fileName = filepath.Join(o.Dir, v1alpha1.SecretMappingFileName)
	}
	o.SecretMapping = *secretMapping

	// lets re-parse the CLI arguments to re-populate the loaded requirements
	err = o.Cmd.Flags().Parse(os.Args)
	if err != nil {
		return errors.Wrap(err, "failed to reparse arguments")
	}

	err = o.applyDefaults()
	if err != nil {
		return err
	}

	err = o.SecretMapping.SaveConfig(fileName)
	if err != nil {
		return errors.Wrapf(err, "failed to save %s", fileName)
	}

	log.Logger().Debugf("saved file: %s", termcolor.ColorInfo(fileName))
	return nil
}

func (o *Options) applyDefaults() error {
	s := &o.SecretMapping

	var err error
	if s.Spec.Defaults.BackendType == v1alpha1.BackendTypeGSM {
		s.Spec.Defaults.GcpSecretsManager, err = o.applyGSMDefaults(s.Spec.Defaults.GcpSecretsManager)
		if err != nil {
			return errors.Wrapf(err, "failed to apply defaults to GcpSecretsManager")
		}
	}

	for k := range s.Spec.Secrets {
		secret := &s.Spec.Secrets[k]
		if secret.BackendType == v1alpha1.BackendTypeGSM {
			secret.GcpSecretsManager, err = o.applyGSMDefaults(secret.GcpSecretsManager)
			if err != nil {
				return errors.Wrapf(err, "failed to apply defaults to GcpSecretsManager for secret %s", secret.Name)
			}
		}
	}

	if s.Spec.Defaults.BackendType == v1alpha1.BackendTypeAzure {
		s.Spec.Defaults.AzureKeyVaultConfig, err = o.applyAzureKeyVaultDefaults(s.Spec.Defaults.AzureKeyVaultConfig)
		if err != nil {
			return errors.Wrapf(err, "failed to apply defaults to Azure Key Vault")
		}
	}

	for k := range s.Spec.Secrets {
		secret := &s.Spec.Secrets[k]
		if secret.BackendType == v1alpha1.BackendTypeAzure {
			secret.AzureKeyVaultConfig, err = o.applyAzureKeyVaultDefaults(secret.AzureKeyVaultConfig)
			if err != nil {
				return errors.Wrapf(err, "failed to apply defaults to Azure Key Vault for secret %s", secret.Name)
			}
		}
	}
	return nil
}

func (o *Options) applyGSMDefaults(gsmConfig *v1alpha1.GcpSecretsManager) (*v1alpha1.GcpSecretsManager, error) {
	if gsmConfig == nil {
		gsmConfig = &v1alpha1.GcpSecretsManager{}
	}
	if gsmConfig.ProjectID == "" {
		if o.requirements.Cluster.ProjectID == "" {
			return gsmConfig, errors.New("found an empty gcp project id and no requirements.Cluster.ProjectID")
		}
		gsmConfig.ProjectID = o.requirements.Cluster.ProjectID
	}
	if gsmConfig.UniquePrefix == "" {
		if o.requirements.Cluster.ClusterName == "" {
			return gsmConfig, errors.New("found an empty gcp project id and no requirements.Cluster.ClusterName")
		}
		gsmConfig.UniquePrefix = o.requirements.Cluster.ClusterName
	}
	if gsmConfig.Version == "" {
		gsmConfig.Version = "latest"
	}
	return gsmConfig, nil
}

func (o *Options) applyAzureKeyVaultDefaults(akvConfig *v1alpha1.AzureKeyVaultConfig) (*v1alpha1.AzureKeyVaultConfig, error) {
	if akvConfig == nil {
		akvConfig = &v1alpha1.AzureKeyVaultConfig{}
	}
	if akvConfig.KeyVaultName == "" {
		if o.requirements.Cluster.AzureConfig == nil || o.requirements.Cluster.AzureConfig.AzureSecretStorageConfig == nil || o.requirements.Cluster.AzureConfig.AzureSecretStorageConfig.KeyVaultName == "" {
			return akvConfig, errors.New("found an empty Azure Key Vault name and one is required to store secrets in Azure Key Vault")
		}
		akvConfig.KeyVaultName = o.requirements.Cluster.AzureConfig.AzureSecretStorageConfig.KeyVaultName
	}
	return akvConfig, nil
}
