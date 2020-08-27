package factory

import (
	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner"
	v1 "github.com/jenkins-x/jx-secret/pkg/apis/external/v1"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets/editor"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets/editor/gsm"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets/editor/local"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets/editor/vault"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
)

func NewEditor(cache map[string]editor.Interface, secret *v1.ExternalSecret, commandRunner cmdrunner.CommandRunner, client kubernetes.Interface) (editor.Interface, error) {
	backendType := secret.Spec.BackendType
	switch backendType {
	case "local":
		return local.NewEditor(client, secret)
	case "vault":
		// lets cache vault editors
		cached := cache["vault"]
		var err error
		if cached == nil {
			cached, err = vault.NewEditor(commandRunner, client)
			if err != nil {
				return cached, errors.Wrapf(err, "failed to create vault editor")
			}
			cache["vault"] = cached
		}
		return cached, nil
	case "gcpSecretsManager":
		return gsm.NewEditor(commandRunner, client)
	default:
		return nil, errors.Errorf("unsupported ExternalSecret back end %s", backendType)
	}
}
