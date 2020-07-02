package wait

import (
	"fmt"
	"time"

	"github.com/jenkins-x/jx-helpers/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/pkg/kube/pods"
	"github.com/jenkins-x/jx-helpers/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets"
	"github.com/jenkins-x/jx-secret/pkg/root"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
)

var (
	cmdLong = templates.LongDesc(`
		Waits for vault to be ready for use
`)

	cmdExample = templates.Examples(`
		%s vault wait
	`)
)

// Options the options for the command
type Options struct {
	PodName      string
	Namespace    string
	Console      bool
	PollDuration time.Duration
	KubeClient   kubernetes.Interface
}

// NewCmdWait creates a command object for the command
func NewCmdWait() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "wait",
		Short:   "Waits for vault to be ready for use",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, root.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	o.AddFlags(cmd)
	return cmd, o
}

// AddFlags adds the options flags to the command
func (o *Options) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&o.PodName, "pod", "p", "vault-0", "the name of the vault pod which needs to be running before the port forward can take place")
	cmd.Flags().StringVarP(&o.Namespace, "ns", "n", "vault-infra", "the namespace where vault is running")
	cmd.Flags().DurationVarP(&o.PollDuration, "duration", "d", 5*time.Minute, "the maximum time period to wait for the vault pod to be ready")
}

// Run implements the command
func (o *Options) Run() error {
	var err error
	o.KubeClient, err = extsecrets.LazyCreateKubeClient(o.KubeClient)
	if err != nil {
		return errors.Wrap(err, "failed to create kube client")
	}
	err = o.waitForPod()
	if err != nil {
		return errors.Wrapf(err, "failed to wait for vault pod")
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
