package konghcl

import (
	"fmt"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/require"
)

type TestSample struct {
	Name string `hcl:"name"`
	Game string `hcl:"game"`
}

func TestHCLFileMapper(t *testing.T) {
	var cli struct {
		Sample TestSample `type:"hclfile"`
	}
	opt := kong.NamedMapper("hclfile", HCLFileMapper)
	parser, err := kong.New(&cli, opt)
	require.NoError(t, err)

	_, err = parser.Parse([]string{"--sample", "testdata/sample.hcl"})
	require.NoError(t, err)

	want := TestSample{Name: "Lee Sedol", Game: "Go"}
	require.Equal(t, want, cli.Sample)
}

func TestHCLFileMapperErr(t *testing.T) {
	var cli struct {
		Sample TestSample `type:"hclfile"`
	}
	opts := []kong.Option{
		kong.NamedMapper("hclfile", HCLFileMapper),
		kong.Exit(func(int) { fmt.Println("EXIT") }),
	}
	parser, err := kong.New(&cli, opts...)
	require.NoError(t, err)

	_, err = parser.Parse([]string{"--sample", "testdata/MISSING_FILE.hcl"})
	require.Error(t, err)

	_, err = parser.Parse([]string{"--sample"})
	require.Error(t, err)
}
