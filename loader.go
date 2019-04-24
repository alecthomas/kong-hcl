package konghcl

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/hashicorp/hcl"
	"github.com/pkg/errors"
)

// Resolver resolves kong Flags from configuration in HCL.
type Resolver struct {
	config map[string]interface{}
}

var _ kong.ConfigurationLoader = Loader

// DecodeValue decodes Kong values into a Go structure.
func DecodeValue(ctx *kong.DecodeContext, dest interface{}) error {
	v := ctx.Scan.Pop().Value
	var (
		data []byte
		err  error
	)
	switch v := v.(type) {
	case string:
		// Value is a string; it can either be a filename or a HCL fragment.
		if strings.HasPrefix(v, "{") {
			data = []byte(v)
		} else {
			filename := kong.ExpandPath(v)
			data, err = ioutil.ReadFile(filename) // nolint: gosec
			if err != nil {
				return fmt.Errorf("invalid HCL in %q: %s", filename, err)
			}
		}
	case []map[string]interface{}:
		merged := map[string]interface{}{}
		for _, m := range v {
			for k, v := range m {
				merged[k] = v
			}
		}
		data, err = json.Marshal(merged)
		if err != nil {
			return err
		}

	default:
		data, err = json.Marshal(v)
		if err != nil {
			return err
		}
	}
	return errors.Wrapf(hcl.Unmarshal(data, dest), "invalid HCL %q", data)
}

// Loader is a Kong configuration loader for HCL.
func Loader(r io.Reader) (kong.Resolver, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	config := map[string]interface{}{}
	err = hcl.Unmarshal(data, &config)
	if err != nil {
		return nil, errors.Wrap(err, "invalid HCL")
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
			if _, ok := node.Target.Interface().(kong.MapperValue); ok {
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

func (r *Resolver) Resolve(context *kong.Context, parent *kong.Path, flag *kong.Flag) (interface{}, error) { // nolint: golint
	path := r.pathForFlag(parent, flag)
	return find(r.config, path)
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
