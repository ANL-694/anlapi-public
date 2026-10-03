package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildAccountDataPayloadUsesCurrentFormat(t *testing.T) {
	payload := BuildAccountDataPayload(nil, nil, func(string, string, int, string, string) string { return "" })

	require.Equal(t, AccountDataType, payload.Type)
	require.Equal(t, AccountDataVersion, payload.Version)
	require.Empty(t, payload.Accounts)
	require.Empty(t, payload.Proxies)
}
