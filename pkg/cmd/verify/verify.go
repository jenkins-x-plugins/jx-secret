package verify

import (
	"fmt"

	"github.com/jenkins-x/jx-extsecret/pkg/extsecrets"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/jenkins-x/jx-promote/pkg/common"
	"github.com/jenkins-x/jx/v2/pkg/cmd/helper"
	"github.com/jenkins-x/jx/v2/pkg/cmd/templates"
	"github.com/jenkins-x/jx/v2/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	extsecretLong = templates.LongDesc(`
		Verifies that the ExternalSecret resources have the required properties populated in the underlying secret storage
`)

	extsecretExample = templates.Examples(`
		%s verify
	`)
)

// Options the options for the command
type Options struct {
	Client    extsecrets.Interface
	Namespace string
}

// NewCmdVerify creates a command object for the command
func NewCmdVerify() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "verify",
		Short:   "Verifies that the ExternalSecret resources have the required properties populated in the underlying secret storage",
		Long:    extsecretLong,
		Example: fmt.Sprintf(extsecretExample, common.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Namespace, "ns", "n", "", "the namespace to filter the ExternalSecret resources")
	return cmd, o
}

// Run implements the command
func (o *Options) Run() error {
	var err error
	if o.Client == nil {
		o.Client, err = extsecrets.NewClient(nil)
		if err != nil {
			return errors.Wrapf(err, "failed to create an extsecret Client")
		}
	}

	resources, err := o.Client.List(o.Namespace, metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to find external secrets")
	}

	log.Logger().Infof("found %d ExternalSecret resources", len(resources))
	for _, r := range resources {
		log.Logger().Infof("found ExternalSecret %s in namespace %s", util.ColorInfo(r.Name), util.ColorInfo(r.Namespace))
	}
	return nil
}
