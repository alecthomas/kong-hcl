package konghcl

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/hashicorp/hcl"
)

// Resolver resolves kong Flags from configuration in HCL.
type Resolver struct {
	config map[string]interface{}
}

var _ kong.ConfigurationLoader = Loader

// Loader is a Kong configuration loader for HCL.
func Loader(r io.Reader) (kong.Resolver, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	config := map[string]interface{}{}
	err = hcl.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	return &Resolver{config: config}, nil
}

func (r *Resolver) Validate(app *kong.Application) error { // nolint: golint
	// Find all valid configuration keys from the Application.
	valid := map[string]bool{}
	rawPrefixes := []string{}
	path := []string{}
	_ = kong.Visit(app, func(node kong.Visitable, next kong.Next) error {
		switch node := node.(type) {
		case *kong.Node:
			path = append(path, node.Name)
			_ = next(nil)
			path = path[:len(path)-1]
			return nil

		case *kong.Flag:
			flagPath := append([]string{}, path...)
			if node.Group != "" {
				flagPath = append(flagPath, node.Group)
			}
			key := strings.Join(append(flagPath, node.Name), "-")
			if _, ok := node.Value.Target.Interface().(RawConfigFlag); ok {
				rawPrefixes = append(rawPrefixes, key)
			} else {
				valid[key] = true
			}

		default:
			return next(nil)
		}
		return nil
	})
	// Then check all configuration keys against the Application keys.
next:
	for key := range flattenConfig(valid, r.config) {
		if !valid[key] {
			for _, prefix := range rawPrefixes {
				if strings.HasPrefix(key, prefix) {
					continue next
				}
			}
			return fmt.Errorf("unknown configuration key %q", key)
		}
	}
	return nil
}

func (r *Resolver) Resolve(context *kong.Context, parent *kong.Path, flag *kong.Flag) (string, error) { // nolint: golint
	path := r.pathForFlag(parent, flag)
	value, err := find(r.config, path)
	if err != nil {
		return "", err
	}
	// Raw config flags are returned as JSON-encoded objects.
	if _, ok := flag.Value.Target.Interface().(RawConfigFlag); value != nil && ok {
		bare, ok := value.([]map[string]interface{})
		if !ok || len(bare) != 1 {
			return "", fmt.Errorf("expected configuration key %q (for flag %s) to be a single map but got %T",
				strings.Join(path, "-"), flag, value)
		}
		data, err := json.Marshal(bare[0])
		return string(data), err
	}
	return stringify(value)
}

func flattenConfig(schema map[string]bool, config map[string]interface{}) map[string]bool {
	out := map[string]bool{}
next:
	for _, path := range flattenNode(config) {
		for i := len(path) - 1; i >= 0; i-- {
			candidate := strings.Join(path[:i], "-")
			if schema[candidate] {
				out[candidate] = true
				continue next
			}
		}
		out[strings.Join(path, "-")] = true
	}
	return out
}

func flattenNode(config interface{}) [][]string {
	out := [][]string{}
	switch config := config.(type) {
	case []map[string]interface{}:
		for _, group := range config {
			out = append(out, flattenNode(group)...)
		}
	case map[string]interface{}:
		for key, value := range config {
			children := flattenNode(value)
			if len(children) == 0 {
				out = append(out, []string{key})
			} else {
				for _, childValue := range children {
					out = append(out, append([]string{key}, childValue...))
				}
			}
		}

	case []interface{}:
		for _, el := range config {
			out = flattenNode(el)
		}

	case bool, float64, int, string:
		return nil

	default:
		panic(fmt.Sprintf("unsupported type %T", config))
	}
	return out
}

// Build a string path up to this flag.
func (r *Resolver) pathForFlag(parent *kong.Path, flag *kong.Flag) []string {
	path := []string{}
	for n := parent.Node(); n != nil && n.Type != kong.ApplicationNode; n = n.Parent {
		path = append([]string{n.Name}, path...)
	}
	if flag.Group != "" {
		path = append([]string{flag.Group}, path...)
	}
	path = append(path, flag.Name)
	return path
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

	case []map[string]interface{}:
		parts := []string{}
		for _, m := range value {
			for k, v := range m {
				sv, err := stringify(v)
				if err != nil {
					return "", err
				}
				parts = append(parts, fmt.Sprintf("%s=%s", k, sv))
			}
		}
		return kong.JoinEscaped(parts, ';'), nil

	case []interface{}:
		parts := []string{}
		for _, n := range value {
			sn, err := stringify(n)
			if err != nil {
				return "", err
			}
			parts = append(parts, sn)
		}
		return kong.JoinEscaped(parts, ','), nil
	}

	return "", fmt.Errorf("unsupported value of type %T", value)
}
