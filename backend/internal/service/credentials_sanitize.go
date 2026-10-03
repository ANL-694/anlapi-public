package service

// SanitizeStoredCredentials removes temporary login secrets that should not
// remain beside OAuth credentials after account authentication is established.
func SanitizeStoredCredentials(platform string, credentials map[string]any) map[string]any {
	if credentials == nil {
		return nil
	}
	_ = platform
	for _, key := range []string{
		"password", "sso_token", "sso", "sso-rw", "clearTextPassword", "cookie",
	} {
		delete(credentials, key)
	}
	return credentials
}
