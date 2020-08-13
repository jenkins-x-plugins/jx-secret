package secretfacade

import (
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/pkg/errors"
)

func (o *Options) Verify() ([]*SecretPair, error) {
	pairs, err := o.Load()
	if err != nil {
		return nil, errors.Wrap(err, "failed to load ExternalSecret and Secret pairs")
	}
	log.Logger().Debugf("found %d ExternalSecret resources", len(pairs))

	for _, p := range pairs {
		r := p.ExternalSecret
		secret := p.Secret
		name := r.Name
		ns := r.Namespace
		result, err := VerifySecret(&r, secret)
		if err != nil {
			return pairs, errors.Wrapf(err, "failed to verify secret %s in namespace %s", name, ns)
		}
		p.Error = result
	}
	return pairs, nil
}
