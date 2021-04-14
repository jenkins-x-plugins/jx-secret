package copy

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/fields"

	"github.com/jenkins-x-plugins/jx-secret/pkg/extsecrets"
	"github.com/jenkins-x-plugins/jx-secret/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/jxenv"
	"github.com/jenkins-x/jx-helpers/v3/pkg/options"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	cmdLong = templates.LongDesc(`
		Copies secrets with the given selector or name to a destination namespace
`)

	cmdExample = templates.Examples(`
		# copy secrets by label from the current namespace 
		jx secret copy --selector mylabel=cheese --to my-preview-ns

		# copy secrets by name from the current namespace 
		jx secret copy --name my-awesome-secret --to my-preview-ns
	`)
)

// Options the options for the command
type Options struct {
	Namespace       string
	ToNamespace     string
	Selector        string
	Name            string
	CreateNamespace bool
	KubeClient      kubernetes.Interface
}

// NewCmdCopy creates a command object for the command
func NewCmdCopy() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "copy",
		Short:   "Copies secrets with the given selector or name to a destination namespace",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Namespace, "ns", "n", "", "the namespace to filter the Secret resources")
	cmd.Flags().StringVarP(&o.Name, "name", "", "", "the name of the Secret to copy")
	cmd.Flags().StringVarP(&o.ToNamespace, "to", "t", "", "the namespace to copy the secrets to")
	cmd.Flags().StringVarP(&o.Selector, "selector", "l", "", "the label selector to find the secrets to copy")
	cmd.Flags().BoolVarP(&o.CreateNamespace, "create-namespace", "", false, "create the to Namespace if it does not already exist")
	return cmd, o
}

// Run implements the command
func (o *Options) Run() error {
	if o.ToNamespace == "" {
		return options.MissingOption("to")
	}
	if o.Selector == "" && o.Name == "" {
		return options.MissingOption("selector")
	}

	var err error
	o.KubeClient, o.Namespace, err = kube.LazyCreateKubeClientAndNamespace(o.KubeClient, o.Namespace)
	if err != nil {
		return errors.Wrapf(err, "failed to create kube client")
	}

	if o.CreateNamespace {
		err = jxenv.EnsureNamespaceCreated(o.KubeClient, o.ToNamespace, nil, nil)
		if err != nil {
			return errors.Wrapf(err, "failed to create namespace %s", o.ToNamespace)
		}
	}
	ns := o.Namespace
	selector := o.Selector
	listOptions := metav1.ListOptions{
		LabelSelector: selector,
	}
	if o.Name != "" {
		listOptions.FieldSelector = fields.OneTermEqualSelector("metadata.name", o.Name).String()
	}
	secrets, err := o.KubeClient.CoreV1().Secrets(ns).List(context.TODO(), listOptions)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Logger().Warnf("no Secrets in namespace %s with selector %s", ns, selector)
			return nil
		}
		return errors.Wrapf(err, "failed to find Secrets in namespace %s with selector %s", ns, selector)
	}
	for i := range secrets.Items {
		secret := &secrets.Items[i]

		// in unit tests the field selector doesn't tend to work with fake clients so lets add an extra check here...
		if o.Name != "" && secret.Name != o.Name {
			continue
		}
		err = extsecrets.CopySecretToNamespace(o.KubeClient, o.ToNamespace, secret)
		if err != nil {
			return errors.Wrapf(err, "failed to copy secret %s to namespace %s", secret.Name, o.ToNamespace)
		}

		log.Logger().Infof("copied secret  %s to namespace %s", secret.Name, o.ToNamespace)
	}
	return nil
}
