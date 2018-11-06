// nolint: govet
package konghcl

import (
	"io"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/require"
)

func TestRawConfig(t *testing.T) {
	var cli struct {
		Flag     string
		Fragment RawConfigFlag
	}
	r := strings.NewReader(`
	flag = "hello"
	fragment {
		str = "field"
		num = 10
		obj {
			one = 1
			two = 2
		}
	}
	`)
	resolver, err := Loader(r)
	require.NoError(t, err)
	parser, err := kong.New(&cli, kong.Resolvers(resolver))
	require.NoError(t, err)
	_, err = parser.Parse(nil)

	type Frag struct {
		Str string
		Num int
		Obj struct {
			One int
			Two int
		}
	}

	expected := Frag{
		Str: "field",
		Num: 10,
		Obj: struct {
			One int
			Two int
		}{1, 2},
	}

	t.Run("FromFragment", func(t *testing.T) {
		require.NoError(t, err)

		actual := Frag{}
		err = cli.Fragment.UnmarshalHCL(&actual)
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})

	t.Run("FromFile", func(t *testing.T) {
		r := strings.NewReader(`
			str = "field"
			num = 10
			obj {
				one = 1
				two = 2
			}
		`)
		w, err := ioutil.TempFile("", "")
		require.NoError(t, err)
		defer w.Close()
		defer os.Remove(w.Name())
		_, err = io.Copy(w, r)
		require.NoError(t, err)
		w.Close()

		_, err = parser.Parse([]string{"--fragment", w.Name()})
		require.NoError(t, err)

		actual := Frag{}
		err = cli.Fragment.UnmarshalHCL(&actual)
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})
}
