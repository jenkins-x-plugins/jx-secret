package extsecrets

import (
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kyamls"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"
)

var externalSecretFilter = kyamls.Filter{
	Kinds: []string{APIVersion + "/" + esv1.ExtSecretKind},
}

func NewFileClient(dir string) Interface {
	return &fileClient{dir}
}

type fileClient struct {
	dir string
}

func (c *fileClient) List(ns string) ([]*esv1.ExternalSecret, error) {
	rNodes, err := kyamls.Collect(c.dir, externalSecretFilter)
	if err != nil {
		return nil, errors.Wrapf(err, "error retrieving external secrets from dir %s", c.dir)
	}
	var externalSecrets []*esv1.ExternalSecret
	for i := range rNodes {
		esNode := rNodes[i]
		raw, err := esNode.String()
		if err != nil {
			log.Logger().Debugf("ignored file we could not stringify as a kubernetes resource: %s", err.Error())
			continue
		}
		es := &esv1.ExternalSecret{}
		if err := yaml.Unmarshal([]byte(raw), es); err != nil {
			log.Logger().Debugf("ignored file we could not decode as an ExternalSecret: %s", err.Error())
			continue
		}
		if ns == "" || es.Namespace == ns {
			externalSecrets = append(externalSecrets, es)
		}
	}
	return externalSecrets, nil
}
