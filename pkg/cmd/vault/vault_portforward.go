package vault

import (
	"fmt"
	"time"

	"github.com/jenkins-x/jx-extsecret/pkg/extsecrets"
	"github.com/jenkins-x/jx-extsecret/pkg/root"
	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/pkg/kube/pods"
	"github.com/jenkins-x/jx-helpers/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
)

var (
	cmdLong = templates.LongDesc(`
		Runs a port forward process so you can access the vault in a kubernetes cluster
`)

	cmdExample = templates.Examples(`
		%s vault portforward
	`)
)

// Options the options for the command
type Options struct {
	PodName       string
	Namespace     string
	Console       bool
	PollDuration  time.Duration
	KubeClient    kubernetes.Interface
	CommandRunner cmdrunner.CommandRunner
}

// NewCmdPortForward creates a command object for the command
func NewCmdPortForward() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "portforward",
		Short:   "Runs a port forward process so you can access the vault in a kubernetes cluster",
		Aliases: []string{"portfwd", "port-forward"},
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, root.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.PodName, "pod", "p", "vault-0", "the name of the vault pod which needs to be running before the port forward can take place")
	cmd.Flags().StringVarP(&o.Namespace, "ns", "n", "vault-infra", "the namespace where vault is running")
	cmd.Flags().DurationVarP(&o.PollDuration, "duration", "d", 5*time.Second, "the time period between polls for the vault pod being ready")
	return cmd, o
}

// Run implements the command
func (o *Options) Run() error {
	var err error
	o.KubeClient, err = extsecrets.LazyCreateKubeClient(o.KubeClient)
	if err != nil {
		return errors.Wrap(err, "failed to create kube client")
	}
	if o.CommandRunner == nil {
		o.CommandRunner = cmdrunner.DefaultCommandRunner
	}
	err = o.waitForPod()
	if err != nil {
		return errors.Wrapf(err, "failed to wait for vault pod")
	}

	cmd := &cmdrunner.Command{
		Name: "kubectl",
		Args: []string{"port-forward", "--namespace", o.Namespace, "service/vault", "8200"},
	}
	_, err = o.CommandRunner(cmd)
	if err != nil {
		return errors.Wrapf(err, "failed to run command: %s", cmd.CLI())
	}
	return nil
}

func (o *Options) waitForPod() error {
	err := pods.WaitForPodNameToBeReady(o.KubeClient, o.Namespace, o.PodName, o.PollDuration)
	if err != nil {
		return errors.Wrapf(err, "failed to wait for pod %s to be ready in namespace %s", o.PodName, o.Namespace)
	}
	log.Logger().Infof("pod %s in namespace %s is ready", termcolor.ColorInfo(o.PodName), termcolor.ColorInfo(o.Namespace))
	return nil
}
