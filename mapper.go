package konghcl

import (
	"io/ioutil"
	"os"
	"reflect"

	"github.com/alecthomas/kong"
	"github.com/hashicorp/hcl"
)

// HCLFileMapper implements kong.MapperValue to decode an HCL file into
// a struct field.
//
//    var cli struct {
//      Profile Profile `type:"hclfile"`
//    }
//
//    func main() {
//      kong.Parse(&cli, kong.NamedMapper("hclfile", konghcl.HCLFileMapper))
//    }
var HCLFileMapper = kong.MapperFunc(decodeHCLFile) //hsnolint: gochecknoglobals

func decodeHCLFile(ctx *kong.DecodeContext, target reflect.Value) error {
	var fname string
	if err := ctx.Scan.PopValueInto("filename", &fname); err != nil {
		return err
	}
	f, err := os.Open(fname) //nolint:gosec
	if err != nil {
		return err
	}
	defer f.Close() //nolint
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}
	return hcl.Unmarshal(b, target.Addr().Interface())
}
