package secrets

import (
	"github.com/pkg/errors"
	"github.com/sethvargo/go-password/password"
)

const (
	allowedSymbols   = "~!#%^_+-=?,."
	upperCaseAllowed = true
	allowRepeat      = true
	numDigits        = 4
	numSymbols       = 2
)

// DefaultGenerateSecret generates a secret using sensible defaults
func DefaultGenerateSecret(length int) (string, error) {
	input := password.GeneratorInput{
		Symbols: allowedSymbols,
	}

	generator, err := password.NewGenerator(&input)
	if err != nil {
		return "", errors.Wrap(err, "unable to create password generator")
	}

	secret, err := generator.Generate(length, numDigits, numSymbols, !upperCaseAllowed, allowRepeat)

	if err != nil {
		return "", errors.Wrap(err, "unable to generate secret")
	}
	return secret, nil
}
