package service

import (
	"strings"
	"testing"
)

func TestKiroUsageHTTPErrorRedactsOAuthSecrets(t *testing.T) {
	err := (&kiroUsageHTTPError{
		StatusCode: 401,
		Body:       `{"access_token":"access-secret","refresh_token":"refresh-secret","client_secret":"client-secret"}`,
	}).Error()

	for _, secret := range []string{"access-secret", "refresh-secret", "client-secret"} {
		if strings.Contains(err, secret) {
			t.Fatalf("Kiro usage error leaked OAuth secret %q: %s", secret, err)
		}
	}
	if !strings.Contains(err, `"access_token":"***"`) {
		t.Fatalf("Kiro usage error did not redact access token: %s", err)
	}
}
