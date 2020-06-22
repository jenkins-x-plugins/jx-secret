package extsecrets

import (
	"github.com/jenkins-x/jx-extsecret/pkg/apis/extsecret/v1alpha1"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"knative.dev/pkg/apis/duck"
)

// Client an implementation of the interface
type client struct {
	dynamicClient dynamic.Interface
}

func (c *client) List(ns string, listOptions metav1.ListOptions) ([]*v1alpha1.ExternalSecret, error) {
	var client dynamic.ResourceInterface
	if ns != "" {
		client = c.dynamicClient.Resource(ExternalSecretsResource).Namespace(ns)
	} else {
		client = c.dynamicClient.Resource(ExternalSecretsResource)
	}
	resources, err := client.List(listOptions)
	if err != nil && apierrors.IsNotFound(err) {
		err = nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to find external secrets")
	}

	var answer []*v1alpha1.ExternalSecret
	if resources != nil {
		for _, u := range resources.Items {
			extSecret := &v1alpha1.ExternalSecret{}
			err = ToStructured(&u, extSecret)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to convert to ExternalSecret %s", u.GetName())
			}
			answer = append(answer, extSecret)
		}
	}
	return answer, nil
}

// ToStructured converts an unstructured object to a pointer to a structured type
func ToStructured(u *unstructured.Unstructured, structured interface{}) error {
	if err := duck.FromUnstructured(u, structured); err != nil {
		return errors.Wrapf(err, "failed to convert unstructured object to %#v", structured)
	}
	return nil
}
