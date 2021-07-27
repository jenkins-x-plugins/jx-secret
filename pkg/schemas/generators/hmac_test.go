package generators_test

import (
	"github.com/jenkins-x-plugins/jx-secret/pkg/schemas/generators"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestHMAC(t *testing.T) {
	text, err := generators.Hmac(nil)
	require.NoError(t, err, "should not fail")

	require.NotEmpty(t, text, "should have returned text")

	t.Logf("created hmac %s\n", text)
}
