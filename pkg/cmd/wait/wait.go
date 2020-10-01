package wait

import (
	"fmt"
	"strings"
	"time"

	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets/secretfacade"
	"github.com/jenkins-x/jx-secret/pkg/rootcmd"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	cmdLong = templates.LongDesc(`
		Waits for the mandatory Secrets to be populated from their External Secrets
`)

	cmdExample = templates.Examples(`
		%s wait
	`)
)

// Options the options for the command
type Options struct {
	secretfacade.Options

	Timeout       time.Duration
	PollPeriod    time.Duration
	Results       []*secretfacade.SecretError
	messages      map[string]string
	loggedMissing bool
}

// NewCmdWait creates a command object for the command
func NewCmdWait() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "wait",
		Short:   "Waits for the mandatory Secrets to be populated from their External Secrets",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Namespace, "ns", "n", "", "the namespace to filter the ExternalSecret resources")
	cmd.Flags().DurationVarP(&o.Timeout, "timeout", "t", 30*time.Minute, "the maximum amount of time to wait for the secrets to be valid")
	cmd.Flags().DurationVarP(&o.PollPeriod, "poll", "p", 2*time.Second, "the polling period to check if the secrets are valid")
	return cmd, o
}

// Run implements the command
func (o *Options) Run() error {
	endTime := time.Now().Add(o.Timeout)
	log.Logger().Infof("waiting for the mandatory Secrets to be populated from ExternalSecrets...")
	for {
		valid, err := o.WaitCheck()
		if err != nil {
			return err
		}
		if valid {
			return nil
		}
		now := time.Now()
		if now.After(endTime) {
			return errors.Errorf("timed out waiting for the Secrets to be valid from the ExternalSecrets after waiting %s", o.Timeout.String())
		}
		time.Sleep(o.PollPeriod)
	}
}

// Run implements the command
func (o *Options) WaitCheck() (bool, error) {
	pairs, err := o.Load()
	if err != nil {
		return false, errors.Wrap(err, "failed to verify secrets")
	}

	valid := true
	count := 0
	for _, r := range pairs {
		if !o.Matches(r) {
			continue
		}
		count++
		state, err := secretfacade.VerifySecret(&r.ExternalSecret, r.Secret)
		if err != nil {
			return false, errors.Wrapf(err, "failed to verify secret")
		}
		name := r.ExternalSecret.Name

		if state != nil && len(state.EntryErrors) > 0 {
			valid = false
			buf := strings.Builder{}

			for i, e := range state.EntryErrors {
				if i > 0 {
					buf.WriteString(", ")
				}
				buf.WriteString(fmt.Sprintf("key %s missing properties: %s", e.Key, strings.Join(e.Properties, ", ")))
			}
			o.logMessage(name, termcolor.ColorWarning(buf.String()))
		} else {
			o.logMessage(name, termcolor.ColorInfo(fmt.Sprintf("valid: %s", strings.Join(r.ExternalSecret.KeyAndNames(), ", "))))
		}
	}
	if count == 0 {
		if !o.loggedMissing {
			o.loggedMissing = true
			log.Logger().Infof("no mandatory ExternalSecrets found")
		}
		return false, nil
	}
	if valid {
		log.Logger().Infof("%d mandatory secrets are valid", count)
	}
	return valid, nil
}

// Matches returns true if the given secret pair matches the predicate
func (o *Options) Matches(r *secretfacade.SecretPair) bool {
	return r.IsMandatory()
}

// logMessage lets log a message if the message has changed for the given secret name
func (o *Options) logMessage(name, message string) {
	if o.messages == nil {
		o.messages = map[string]string{}
	}
	if o.messages[name] == message {
		return
	}
	o.messages[name] = message
	log.Logger().Infof("%s: %s", name, message)
}
