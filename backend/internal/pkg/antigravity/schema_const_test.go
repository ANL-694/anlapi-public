package antigravity

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildToolsPreservesStringConstIntersection(t *testing.T) {
	for _, tc := range []struct {
		name   string
		schema string
		want   string
	}{
		{"const only", `{"type":"string","const":"browser"}`, `{"type":"string","enum":["browser"]}`},
		{"inferred const only", `{"const":"browser"}`, `{"type":"string","enum":["browser"]}`},
		{"matching enum", `{"type":"string","const":"browser","enum":["browser","shell"]}`, `{"type":"string","enum":["browser"]}`},
		{"conflicting enum", `{"type":"string","const":"browser","enum":["shell"]}`, `{"type":"string","enum":[]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var property map[string]any
			require.NoError(t, json.Unmarshal([]byte(tc.schema), &property))
			tools := buildTools([]ClaudeTool{{
				Name: "dispatch",
				InputSchema: map[string]any{
					"type":       "object",
					"properties": map[string]any{"action": property},
				},
			}})
			require.Len(t, tools, 1)
			require.Len(t, tools[0].FunctionDeclarations, 1)
			properties, ok := tools[0].FunctionDeclarations[0].Parameters["properties"].(map[string]any)
			require.True(t, ok)
			got, err := json.Marshal(properties["action"])
			require.NoError(t, err)
			require.JSONEq(t, tc.want, string(got))
		})
	}
}
