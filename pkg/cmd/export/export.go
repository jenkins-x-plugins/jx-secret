package export

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-yaml/yaml"
	"github.com/jenkins-x/jx-extsecret/pkg/extsecrets"
	"github.com/jenkins-x/jx-extsecret/pkg/extsecrets/secretfacade"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/jenkins-x/jx-promote/pkg/common"
	"github.com/jenkins-x/jx/v2/pkg/cmd/helper"
	"github.com/jenkins-x/jx/v2/pkg/cmd/templates"
	"github.com/jenkins-x/jx/v2/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	editLong = templates.LongDesc(`
		Exports the current populated values to a YAML file
`)

	editExample = templates.Examples(`
		%s edit
	`)
)

// Options the options for the command
type Options struct {
	secretfacade.Options
	OutFile string
	Console bool
}

// NewCmdExport creates a command object for the command
func NewCmdExport() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "export",
		Short:   "Exports the current populated values to a YAML file",
		Long:    editLong,
		Example: fmt.Sprintf(editExample, common.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Namespace, "ns", "n", "", "the namespace to filter the ExternalSecret resources")
	cmd.Flags().StringVarP(&o.OutFile, "file", "f", "", "the file to use to save the secrets to")
	cmd.Flags().BoolVarP(&o.Console, "console", "c", false, "display the secrets on the console instead of a file")
	return cmd, o
}

// Run implements the command
func (o *Options) Run() error {
	pairs, err := o.Load()
	if err != nil {
		return errors.Wrapf(err, "failed to load ExternalSecret/Secret pairs")
	}

	log.Logger().Debugf("found %d ExternalSecret resources", len(pairs))

	m := map[string]interface{}{}
	for _, p := range pairs {
		if p.Secret == nil || p.Secret.Data == nil {
			continue
		}
		r := p.ExternalSecret
		backendType := r.Spec.BackendType
		for _, data := range r.Spec.Data {
			name := data.Name
			value := p.Secret.Data[name]
			if len(value) == 0 {
				continue
			}

			key := extsecrets.SimplifyKey(backendType, data.Key)
			property := data.Property

			jsonPath := strings.ReplaceAll(key+"/"+property, "/", ".")
			util.SetMapValueViaPath(m, jsonPath, string(value))
		}
	}

	data, err := yaml.Marshal(&m)
	if err != nil {
		return errors.Wrap(err, "failed to marshal secret data to YAML")
	}
	secretsYAML := string(data)

	if o.Console {
		log.Logger().Infof("%s", util.ColorStatus(secretsYAML))
		return nil
	}
	fileName := o.OutFile
	if fileName == "" && !o.Console {
		return util.MissingOption("file")
	}
	if !o.Console {
		dir := filepath.Dir(fileName)
		err := os.MkdirAll(dir, util.DefaultWritePermissions)
		if err != nil {
			return errors.Wrapf(err, "failed to create parent directory %s", dir)
		}
		log.Logger().Debugf("created directory %s", dir)
	}

	err = ioutil.WriteFile(fileName, []byte(secretsYAML), util.DefaultFileWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to save secrets file %s", fileName)
	}
	log.Logger().Infof("exported Secrets to file: %s", util.ColorInfo(fileName))
	return nil
}
