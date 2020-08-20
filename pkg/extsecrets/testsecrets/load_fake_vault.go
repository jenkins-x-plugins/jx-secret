package testsecrets

import (
	"strings"
	"testing"

	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner"
	"github.com/stretchr/testify/assert"
)

type FakeVaultSecrets struct {
	Objects map[string]map[string]string
}

func LoadFakeVaultSecrets(t *testing.T, cmds []*cmdrunner.Command, vaultBin string) FakeVaultSecrets {
	s := FakeVaultSecrets{
		Objects: map[string]map[string]string{},
	}

	for _, cmd := range cmds {
		args := cmd.Args
		if cmd.Name == vaultBin && len(args) > 3 && args[0] == "kv" && args[1] == "put" {
			objectName := args[2]

			if s.Objects[objectName] == nil {
				s.Objects[objectName] = map[string]string{}
			}
			for i := 3; i < len(args); i++ {
				value := args[i]
				if value != "" {
					paths := strings.SplitN(value, "=", 2)
					if len(paths) == 2 {
						k := paths[0]
						v := paths[1]

						t.Logf("secret %s has %s=%s\n", objectName, k, v)
						s.Objects[objectName][k] = v
					}
				}
			}
		}
	}
	return s
}

// AssertHasValue asserts the given property has a value populated
func (s *FakeVaultSecrets) AssertHasValue(t *testing.T, objectName, propertyName string) string {
	object := s.Objects[objectName]
	if assert.NotNil(t, object, "no object found for %s", objectName) {
		value := object[propertyName]
		if assert.NotEmpty(t, value, "no property value for secret %s property %s", objectName, propertyName) {
			t.Logf("secret %s has expected property %s=%s\n", objectName, propertyName, value)
		}
		return value
	}
	return ""
}

// AssertValueEquals asserts the given property value
func (s *FakeVaultSecrets) AssertValueEquals(t *testing.T, objectName, propertyName, expectedValue string) {
	object := s.Objects[objectName]
	if assert.NotNil(t, object, "no object found for %s", objectName) {
		value := object[propertyName]
		assert.Equal(t, expectedValue, value, "property value for secret %s property %s", objectName, propertyName)
		t.Logf("secret %s has expected property %s=%s\n", objectName, propertyName, value)
	}
}

