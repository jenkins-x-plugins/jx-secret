package verify

import (
	"github.com/jenkins-x/jx-extsecret/pkg/extsecrets"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (o *Options) Verify() ([]*SecretError, error) {
	var answer []*SecretError
	var err error

	if o.SecretClient == nil {
		o.SecretClient, err = extsecrets.NewClient(nil)
		if err != nil {
			return answer, errors.Wrapf(err, "failed to create an extsecret Client")
		}
	}
	o.KubeClient, err = extsecrets.LazyCreateKubeClient(o.KubeClient)
	if err != nil {
		return answer, errors.Wrapf(err, "failed to create kube Client")
	}

	resources, err := o.SecretClient.List(o.Namespace, metav1.ListOptions{})
	if err != nil {
		return answer, errors.Wrap(err, "failed to find external secrets")
	}
	o.ExternalSecrets = resources

	log.Logger().Debugf("found %d ExternalSecret resources", len(resources))

	for _, r := range resources {
		ns := r.Namespace
		if ns == "" {
			ns = o.Namespace
		}
		name := r.Name
		if ns == "" {
			log.Logger().Warnf("no namespace found for ExternalSecret %s", name)
			continue
		}

		secret, err := o.KubeClient.CoreV1().Secrets(ns).Get(name, metav1.GetOptions{})
		if err != nil && apierrors.IsNotFound(err) {
			err = nil
		}
		if err != nil {
			return answer, errors.Wrapf(err, "failed to find Secret %s in namespace %s", name, ns)
		}

		result, err := VerifySecret(r, secret)
		if err != nil {
			return answer, errors.Wrapf(err, "failed to verify secret %s in namespace %s", name, ns)
		}
		if result != nil {
			answer = append(answer, result)
		}
	}
	return answer, nil
}
