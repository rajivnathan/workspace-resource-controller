package templates

import (
	"embed"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

//go:embed *.yaml
var embeddedResources embed.FS

// templateArgs represents the arguments to fill in the template
type templateArgs struct {
	Namespace string
	VideoGame string
}

func TestNewSyncerYAML(t *testing.T) {
	expectedYAML := `---
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  namespace: default
type: Opaque
data:
  video_game: mario
`

	resourceTemplate, err := embeddedResources.ReadFile("test.yaml")
	require.NoError(t, err)

	actualYAML, err := RenderResources(resourceTemplate, templateArgs{
		Namespace: "default",
		VideoGame: "mario",
	})
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(expectedYAML, string(actualYAML)))
}
