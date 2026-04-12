package extsecrets

import (
	v1 "github.com/jenkins-x-plugins/jx-secret/pkg/apis/external/v1"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kyamls"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
)

var externalSecretFilter = kyamls.Filter{
	Kinds: []string{"external-secrets.io/v1beta1/ExternalSecret"},
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
		doc := esNode.Document()
		err = doc.Decode(es)
		if err != nil {
			log.Logger().Debugf("ignored file we could not decode it as a kubernetes resource: %s", err.Error())
			continue
		}
		if ns == "" || es.ObjectMeta.Namespace == ns {
			externalSecrets = append(externalSecrets, es)
		}
	}
	return externalSecrets, nil
}
