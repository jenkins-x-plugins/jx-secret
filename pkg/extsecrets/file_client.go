package extsecrets

import (
	"github.com/jenkins-x/jx-helpers/v3/pkg/kyamls"
	v1 "github.com/jenkins-x/jx-secret/pkg/apis/external/v1"
	"github.com/pkg/errors"
)

var externalSecretFilter = kyamls.Filter{
	Kinds: []string{"kubernetes-client.io/v1/ExternalSecret"},
}

func NewFileClient(dir string) Interface {
	return &fileClient{dir}
}

type fileClient struct {
	dir string
}

func (c *fileClient) List(ns string) ([]*v1.ExternalSecret, error) {
	rNodes, err := kyamls.Collect(c.dir, externalSecretFilter)
	if err != nil {
		return nil, errors.Wrapf(err, "error retrieving external secrets from dir %s", c.dir)
	}
	var externalSecrets []*v1.ExternalSecret
	for _, esNode := range rNodes {

		es := &v1.ExternalSecret{}
		err = esNode.Document().Decode(es)
		if err != nil {
			return nil, errors.Wrapf(err, "error decoding external secret in dir %s", c.dir)
		}
		if ns == "" || es.ObjectMeta.Namespace == ns {
			externalSecrets = append(externalSecrets, es)
		}
	}
	return externalSecrets, nil
}
