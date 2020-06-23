package fake

import "github.com/pkg/errors"

// FakeInput provide a fake provider for testing
type FakeInput struct {
	// Values the values to return indexed by the message
	Values map[string]string
}

// PickPassword gets a password (via hidden input) from a user's free-form input
func (f *FakeInput) PickPassword(message string, help string) (string, error) {
	if f.Values == nil {
		f.Values = map[string]string{}
	}
	value := f.Values[message]
	if value == "" {
		return "", errors.Errorf("missing fake value for message: %s", message)
	}
	return value, nil
}
