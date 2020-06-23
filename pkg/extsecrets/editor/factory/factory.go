package factory

import (
	"github.com/jenkins-x/jx-extsecret/pkg/apis/extsecret/v1alpha1"
	"github.com/jenkins-x/jx-extsecret/pkg/cmdrunner"
	"github.com/jenkins-x/jx-extsecret/pkg/extsecrets/editor"
	"github.com/jenkins-x/jx-extsecret/pkg/extsecrets/editor/vault"
	"github.com/pkg/errors"
)

func NewEditor(secret *v1alpha1.ExternalSecret, commandRunner cmdrunner.CommandRunner) (editor.Interface, error) {
	backendType := secret.Spec.BackendType
	switch backendType {
	case "vault":
		return vault.NewEditor(commandRunner)
	default:
		return nil, errors.Errorf("unsupported ExternalSecret back end %s", backendType)
	}
}
