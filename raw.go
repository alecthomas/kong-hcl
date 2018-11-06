package konghcl

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/hashicorp/hcl"
)

// RawConfigFlag is a Kong flag value that can be used to load HCL config either
// from an external file or from fragments embedded in the main configuration
// file.
//
// Use it like so:
//
// 		var cli struct {
// 			Fragment RawConfigFlag
// 		}
// 		// parsing...
//
// 		// Once flags are parsed, unmarshal HCL fragment into an object.
// 		if cli.Fragment != nil {
// 			err = cli.Fragment.UnmarshalHCL(&obj)
// 		}
type RawConfigFlag map[string]interface{}

// UnmarshalHCL unmarshals the flag into a Go value.
func (r *RawConfigFlag) UnmarshalHCL(v interface{}) error {
	// This round-trip is not ideal.
	data, err := json.Marshal(*r)
	if err != nil {
		return err
	}
	return hcl.Decode(v, string(data))
}

// Decode file or fragment into value.
func (r *RawConfigFlag) Decode(ctx *kong.DecodeContext) error {
	var (
		param = ctx.Scan.PopValue("filename")
		data  []byte
		err   error
	)

	if strings.HasPrefix(param, "{") {
		data = []byte(param)
	} else {
		filename := kong.ExpandPath(param)
		data, err = ioutil.ReadFile(filename) // nolint: gosec
		if err != nil {
			return fmt.Errorf("invalid HCL in %q: %s", filename, err)
		}
	}
	err = hcl.Unmarshal(data, r)
	if err != nil {
		return fmt.Errorf("invalid HCL fragment: %s", err)
	}
	return nil
}
