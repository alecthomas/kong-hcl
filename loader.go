package konghcl

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/hashicorp/hcl"
)

// Loader is a Kong configuration loader for HCL.
func Loader(r io.Reader) (kong.ResolverFunc, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	config := map[string]interface{}{}
	err = hcl.Unmarshal(data, &config)
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

		value, err := find(config, path)
		if err != nil {
			return "", err
		}
		return stringify(value)
	}, nil
}

func find(config map[string]interface{}, path []string) (interface{}, error) {
	if len(path) == 0 {
		return nil, nil
	}

	// Check if we have a "prefix-<key>".
	parts := strings.SplitN(path[0], "-", 2)
	if config[path[0]] == nil && len(parts) == 2 {
		if children, ok := config[parts[0]].([]map[string]interface{}); ok {
			path = append([]string{parts[1]}, path[1:]...)
			return find(children[0], path)
		}
	}

	if len(path) == 1 {
		return config[path[0]], nil
	}

	child, ok := config[path[0]]
	if !ok {
		return nil, nil
	}

	if children, ok := child.([]map[string]interface{}); ok {
		return find(children[0], path[1:])
	}

	return nil, nil
}

func stringify(value interface{}) (string, error) {
	switch value := value.(type) {
	case nil:
		return "", nil

	case bool, int, float64:
		return fmt.Sprintf("%v", value), nil

	case string:
		return value, nil

	case []interface{}:
		parts := []string{}
		for _, n := range value {
			sn, err := stringify(n)
			if err != nil {
				return "", err
			}
			parts = append(parts, sn)
		}
		return strings.Join(parts, ","), nil
	}

	return "", fmt.Errorf("invalid value %#v", value)
}
