package konghcl

import (
	"io"

	"github.com/alecthomas/kong"
)

// Loader is a Kong configuration loader for HCL.
func Loader(r io.Reader) (kong.ResolverFunc, error) {
	config := &Config{}
	err := Parser.Parse(r, config)
	if err != nil {
		return nil, err
	}

	return func(context *kong.Context, parent *kong.Path, flag *kong.Flag) (string, error) {
		// Build a string path up to this flag.
		path := []string{}
		for n := parent.Node(); n != nil && n.Type != kong.ApplicationNode; n = n.Parent {
			path = append([]string{n.Name}, path...)
		}
		if flag.Group != "" {
			path = append([]string{flag.Group}, path...)
		}
		path = append(path, flag.Name)

		// Find the HCL value corresponding to this flag path.
		value := config.Find(path)
		if value == nil {
			return "", nil
		}
		return value.String(), nil
	}, nil
}
