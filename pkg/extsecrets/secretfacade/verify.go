package secretfacade

import (
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/jenkins-x/jx-secret/pkg/secretmapping"
	"github.com/pkg/errors"
)

// Verify loads the secrets and verifies which are valid to aid the edit/populate operations
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

// VerifyAndFilter loads the secrets and verifies which are valid to aid the edit/populate operations
// then filters out any duplicate entries which are using the same underlying secret mappings.
//
// e.g. if 2 secrets are populated to the same actual location then we can omit one of them since there's no need
// to write to the same location twice.
//
// We prefer the secrets which have schemas associated and that have the most entries.
func (o *Options) VerifyAndFilter() ([]*SecretPair, error) {
	secrets, err := o.Verify()
	if err != nil {
		return secrets, err
	}

	// lets filter out any secrets with similar secret mapping locations...
	secretMapping, _, err := secretmapping.LoadSecretMapping(o.Dir, false)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load secret mappings in dir %s", o.Dir)
	}

	if secretMapping != nil {
		for _, sm := range secretMapping.Spec.Secrets {
			if sm.Name == "foo" {
				return nil, errors.Errorf("TODO")
			}
		}
	}

	return secrets, nil
}
