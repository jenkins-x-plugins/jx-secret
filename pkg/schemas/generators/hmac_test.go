package generators_test

import (
	"testing"

	"github.com/jenkins-x-plugins/jx-secret/pkg/schemas/generators"
	"github.com/stretchr/testify/require"
)

func TestHMAC(t *testing.T) {
	text, err := generators.Hmac(nil)
	require.NoError(t, err, "should not fail")

	require.NotEmpty(t, text, "should have returned text")

	t.Logf("created hmac %s\n", text)
}
