package generators

import (
	"github.com/jenkins-x/jx-helpers/pkg/stringhelpers"
	"github.com/pkg/errors"
)

// Hmac generates a hmac value for webhooks
func Hmac(args *Arguments) (string, error) {
	value, err := stringhelpers.RandStringBytesMaskImprSrc(41)
	if err != nil {
		return value, errors.Wrapf(err, "generating hmac")
	}
	return value, nil
}
