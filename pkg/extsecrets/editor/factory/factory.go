package factory

import (
	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner"
	"github.com/jenkins-x/jx-secret/pkg/apis/extsecret/v1alpha1"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets/editor"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets/editor/vault"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
)

func NewEditor(secret *v1alpha1.ExternalSecret, commandRunner cmdrunner.CommandRunner, client kubernetes.Interface) (editor.Interface, error) {
	backendType := secret.Spec.BackendType
	switch backendType {
	case "vault":
		return vault.NewEditor(commandRunner, client)
	default:
		return nil, errors.Errorf("unsupported ExternalSecret back end %s", backendType)
	}
}
