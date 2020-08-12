package secretfacade

import (
	"github.com/jenkins-x/jx-helpers/pkg/kube"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Load loads the secret pairs
func (o *Options) Load() ([]*SecretPair, error) {
	var answer []*SecretPair
	var err error

	if o.SecretClient == nil {
		o.SecretClient, err = extsecrets.NewClient(nil)
		if err != nil {
			return answer, errors.Wrapf(err, "failed to create an extsecret Client")
		}
	}
	o.KubeClient, o.Namespace, err = kube.LazyCreateKubeClientAndNamespace(o.KubeClient, o.Namespace)
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
		if r == nil {
			continue
		}
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
		answer = append(answer, &SecretPair{
			ExternalSecret: *r,
			Secret:         secret,
		})
	}
	return answer, nil
}
