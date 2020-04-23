package konghcl

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/pkg/errors"
	"github.com/zclconf/go-cty/cty"
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
	filename := "config.hcl"
	switch v := v.(type) {
	case string:
		// Value is a string; it can either be a filename or a HCL fragment.
		filename = kong.ExpandPath(v)
		data, err = ioutil.ReadFile(filename) // nolint: gosec
		if os.IsNotExist(err) {
			data = []byte(v)
		} else if err != nil {
			return errors.Wrapf(err, "invalid HCL in %q", filename)
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

	parser := hclparse.NewParser()
	var (
		ast  *hcl.File
		diag hcl.Diagnostics
	)
	if bytes.HasPrefix(data, []byte("{")) {
		ast, diag = parser.ParseJSON(data, filename)
	} else {
		ast, diag = parser.ParseHCL(data, filename)
	}
	if diag.HasErrors() {
		return errors.Errorf("invalid HCL %s: %s", data, diag[0].Summary)
	}
	diag = gohcl.DecodeBody(ast.Body, nil, dest)
	if diag.HasErrors() {
		return errors.Errorf("invalid HCL %s: %s", data, diag[0].Summary)
	}
	return nil
}

// Loader is a Kong configuration loader for HCL.
func Loader(r io.Reader) (kong.Resolver, error) {
	filename := "config.hcl"
	if named, ok := r.(interface{ Name() string }); ok {
		filename = named.Name()
	}
	parser := hclparse.NewParser()
	source, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	ast, diag := parser.ParseHCL(source, filename)
	if diag.HasErrors() {
		return nil, errors.Wrap(diag, filename)
	}
	config := map[string]interface{}{}
	err = flattenHCL(nil, ast.Body.(*hclsyntax.Body), config)
	if err != nil {
		return nil, err
	}
	return &Resolver{config: config}, nil
}

func flattenHCL(key []string, node hclsyntax.Node, dest map[string]interface{}) (err error) {
	defer func() {
		if err != nil && len(key) > 0 {
			err = errors.Wrap(err, key[len(key)-1])
		}
	}()
	switch node := node.(type) {
	case hclsyntax.Attributes:
		for attr, value := range node {
			if err := flattenHCL(append(key, attr), value, dest); err != nil {
				return err
			}
		}
	case *hclsyntax.Attribute:
		value, err := decodeHCLExpr(node.Expr)
		if err != nil {
			return err
		}
		dest[strings.Join(key, "-")] = value
	case hclsyntax.Blocks:
		for _, block := range node {
			if err := flattenHCL(key, block, dest); err != nil {
				return err
			}
		}
	case *hclsyntax.Block:
		sub := map[string]interface{}{}
		key = append(key, node.Type)
		for _, label := range node.Labels {
			next := map[string]interface{}{}
			sub[label] = []map[string]interface{}{next}
			sub = next
		}
		if err := flattenHCL(nil, node.Body, sub); err != nil {
			return err
		}
		dkey := strings.Join(key, "-")
		switch value := dest[dkey].(type) {
		case nil:
			dest[dkey] = []map[string]interface{}{sub}
		case []map[string]interface{}:
			value = append(value, sub)
			dest[dkey] = value
		}
	case *hclsyntax.Body:
		if err := flattenHCL(key, node.Attributes, dest); err != nil {
			return err
		}
		if err := flattenHCL(key, node.Blocks, dest); err != nil {
			return err
		}
	default:
		panic(fmt.Sprintf("%T", node))
	}
	return nil
}

func decodeHCLExpr(expr hclsyntax.Expression) (interface{}, error) {
	value, diag := expr.Value(nil)
	if diag.HasErrors() {
		return nil, errors.WithStack(diag)
	}
	return decodeCTYValue(value), nil
}

func decodeCTYValue(value cty.Value) interface{} {
	switch value.Type() {
	case cty.String:
		return value.AsString()
	case cty.Bool:
		return value.True()
	case cty.Number:
		f, _ := value.AsBigFloat().Float64()
		return f
	default:
		if value.Type().IsListType() || value.Type().IsTupleType() {
			out := []interface{}{}
			value.ForEachElement(func(key cty.Value, val cty.Value) (stop bool) {
				out = append(out, decodeCTYValue(val))
				return false
			})
			return out
		} else if value.Type().IsMapType() || value.Type().IsObjectType() {
			out := map[string]interface{}{}
			value.ForEachElement(func(key cty.Value, val cty.Value) (stop bool) {
				out[key.AsString()] = decodeCTYValue(val)
				return false
			})
			return out
		}
	}
	panic(value.Type().GoString())
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
			return errors.Errorf("unknown configuration key %q", key)
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

// Find the value that path maps to.
func find(config map[string]interface{}, path []string) (interface{}, error) {
	if len(path) == 0 {
		return config, nil
	}

	key := strings.Join(path, "-")
	parts := strings.SplitN(key, "-", -1)
	for i := len(parts); i > 0; i-- {
		prefix := strings.Join(parts[:i], "-")
		if sub := config[prefix]; sub != nil {
			if sub, ok := sub.([]map[string]interface{}); ok {
				if len(sub) > 1 {
					return sub, nil
				}
				return find(sub[0], parts[i:])
			}
			return sub, nil
		}
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
