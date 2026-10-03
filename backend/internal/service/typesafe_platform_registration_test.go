package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTypeSafePlatformIsConcreteForRoutingAndComposite(t *testing.T) {
	require.Equal(t, "typesafe", PlatformTypeSafe)
	require.True(t, isConcreteRequestPlatform(PlatformTypeSafe))
}
