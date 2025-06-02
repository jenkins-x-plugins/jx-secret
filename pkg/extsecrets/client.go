package extsecrets

import (
	"context"
	"strings"

	"github.com/jenkins-x/jx-helpers/v3/pkg/knative_pkg/duck"

	v1 "github.com/jenkins-x-plugins/jx-secret/pkg/apis/external/v1"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// Client an implementation of the interface
type client struct {
	dynamicClient dynamic.Interface
	kubeClient    kubernetes.Interface
}

//nolint:gocritic
func (c *client) List(ns string) ([]*v1.ExternalSecret, error) {
	var client dynamic.ResourceInterface
	apiResources, err := c.kubeClient.Discovery().ServerResources()
	legacykes := true
	if err != nil {
		return nil, errors.Wrap(err, "failed to list server resources")
	}

	for k := range apiResources {
		if strings.Contains(apiResources[k].GroupVersion, "external-secrets.io") {
			for k1 := range apiResources[k].APIResources {
				if strings.Contains(apiResources[k].APIResources[k1].Name, "externalsecrets") {
					legacykes = false
				}
			}
		}
	}

	if legacykes {
		if ns != "" {
			client = c.dynamicClient.Resource(KubernetesExternalSecretsResource).Namespace(ns)
		} else {
			client = c.dynamicClient.Resource(KubernetesExternalSecretsResource)
		}
	} else {
		if ns != "" {
			client = c.dynamicClient.Resource(ExternalSecretsResource).Namespace(ns)
		} else {
			client = c.dynamicClient.Resource(ExternalSecretsResource)
		}
	}
	resources, err := client.List(context.TODO(), metav1.ListOptions{})
	if err != nil && apierrors.IsNotFound(err) {
		err = nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to find external secrets")
	}

	var answer []*v1.ExternalSecret
	if resources != nil {
		for k := range resources.Items {
			u := resources.Items[k]
			extSecret := &v1.ExternalSecret{}
			err = FromUnstructured(&u, extSecret)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to convert to ExternalSecret %s", u.GetName())
			}
			answer = append(answer, extSecret)
		}
	}
	return answer, nil
}

// FromUnstructured converts from an unstructured object to a pointer to a structured type
func FromUnstructured(u *unstructured.Unstructured, structured interface{}) error {
	if err := duck.FromUnstructured(u, structured); err != nil {
		return errors.Wrapf(err, "failed to convert unstructured object to %#v", structured)
	}
	return nil
}

// ToStructured converts a resource to an unstructured type
func ToStructured(resource duck.OneOfOurs) (*unstructured.Unstructured, error) {
	u, err := duck.ToUnstructured(resource)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to convert resource %#v to Unstructured", resource)
	}
	return u, nil
}
