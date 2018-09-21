// nolint: govet
package konghcl

import (
	"strings"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/require"
)

func TestHCL(t *testing.T) {
	type Embedded struct {
		EmbeddedFlag string
	}
	var cli struct {
		FlagName     string
		IntFlag      int
		FloatFlag    float64
		SliceFlag    []int
		GroupedFlag  string `group:"group"`
		PrefixedFlag string `prefix:"prefix-"`
		Embedded     `group:"group"`
	}
	r := strings.NewReader(`

		flag-name = "hello world"
		int-flag = 10
		float-flag = 10.5
		slice-flag = [1, 2, 3]

		prefix {
			prefixed-flag = "prefixed flag"
		}
		group {
			grouped-flag = "grouped flag"
			embedded-flag = "embedded flag"
		}

	`)
	resolver, err := Loader(r)
	require.NoError(t, err)
	parser, err := kong.New(&cli, kong.Resolvers(resolver))
	require.NoError(t, err)
	_, err = parser.Parse(nil)
	require.NoError(t, err)
	require.Equal(t, "hello world", cli.FlagName)
	require.Equal(t, "grouped flag", cli.GroupedFlag)
	require.Equal(t, "prefixed flag", cli.PrefixedFlag)
	require.Equal(t, "embedded flag", cli.EmbeddedFlag)
	require.Equal(t, 10, cli.IntFlag)
	require.Equal(t, 10.5, cli.FloatFlag)
	require.Equal(t, []int{1, 2, 3}, cli.SliceFlag)
}

func TestHCLValidation(t *testing.T) {
	type command struct {
		CommandFlag string
	}
	var cli struct {
		Command command `cmd:""`
		Flag    string
	}
	resolver, err := Loader(strings.NewReader(`
		invalid-flag = true
	`))
	require.NoError(t, err)
	parser, err := kong.New(&cli, kong.Resolvers(resolver))
	require.NoError(t, err)
	_, err = parser.Parse([]string{"command"})
	require.EqualError(t, err, "unknown configuration key \"invalid-flag\"")
}
