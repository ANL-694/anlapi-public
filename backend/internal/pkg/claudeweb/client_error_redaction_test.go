package claudeweb

import (
	"io"
	"strings"
	"testing"

	fhttp "github.com/bogdanfinn/fhttp"
)

func TestHTTPErrorRedactsOAuthSecrets(t *testing.T) {
	err := (&HTTPError{
		Operation:  "organizations",
		StatusCode: 401,
		Body:       []byte(`{"access_token":"access-secret","refresh_token":"refresh-secret","client_secret":"client-secret"}`),
	}).Error()

	for _, secret := range []string{"access-secret", "refresh-secret", "client-secret"} {
		if strings.Contains(err, secret) {
			t.Fatalf("Claude Web error leaked OAuth secret %q: %s", secret, err)
		}
	}
	if !strings.Contains(err, `"access_token":"***"`) {
		t.Fatalf("Claude Web error did not redact access token: %s", err)
	}
}

func TestNewHTTPErrorRedactsResponseBodyBeforeForwarding(t *testing.T) {
	response := &fhttp.Response{
		StatusCode: 401,
		Body:       io.NopCloser(strings.NewReader(`{"access_token":"access-secret","refresh_token":"refresh-secret"}`)),
	}
	err := newHTTPError("organizations", response)

	if strings.Contains(string(err.Body), "access-secret") || strings.Contains(string(err.Body), "refresh-secret") {
		t.Fatalf("captured Claude Web body leaked OAuth secret: %s", err.Body)
	}
	if !strings.Contains(string(err.Body), `"access_token":"***"`) {
		t.Fatalf("captured Claude Web body did not redact access token: %s", err.Body)
	}
}
