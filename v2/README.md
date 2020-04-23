# A Kong configuration loader for HCL [![](https://godoc.org/github.com/alecthomas/kong-hcl?status.svg)](http://godoc.org/github.com/alecthomas/kong-hcl/v2) [![CircleCI](https://img.shields.io/circleci/project/github/alecthomas/kong-hcl.svg)](https://circleci.com/gh/alecthomas/kong-hcl)

This is version 2.x of this package which uses the HCL2 library. For most config files it should 
be a drop-in replacement, but for any codebases using `konghcl.DecodeValue()` you will need to
update your Go structs to include [HCL tags](https://pkg.go.dev/github.com/hashicorp/hcl/v2@v2.4.0/gohcl?tab=doc).

Use it like so:

```go
var cli struct {
    Config kong.ConfigFlag `help:"Load configuration."`
}
parser, err := kong.New(&cli, kong.Configuration(konghcl.Loader, "/etc/myapp/config.hcl", "~/.myapp.hcl))
```

## Mapping HCL fragments to a struct

More complex structures can be loaded directly into flag values by implementing the
`kong.MapperValue` interface, and calling `konghcl.DecodeValue`. 

The value can either be a HCL(/JSON) fragment, or a path to a HCL file that will be loaded. Both
can be specified on the command-line or config file.

Note that kong-hcl 2.x uses the HCL2 library, which is *much* stricter about Go tags.
See the [HCL2 documentation](https://pkg.go.dev/github.com/hashicorp/hcl/v2@v2.4.0/gohcl?tab=doc)
for details.

eg.

```go
type NestedConfig struct {
	Size int    `hcl:"size,optional"`
	Name string `hcl:"name,optional"`
}

type ComplexConfig struct {
	Key bool                       `hcl:"key,optional"`
	Nested map[string]NestedConfig `hcl:"nested,optional"`
}

func (c *ComplexConfig) Decode(ctx *kong.DecodeContext) error {
	return konghcl.DecodeValue(ctx, c)
}

// ...

type Config struct {
	Complex ComplexConfig
}
```

Then the following `.hcl` config fragment will be decoded into `Complex`:

```hcl
complex {
  key = true
  nested first {
    size = 10
    name = "first name"
  }
  nested second {
    size = 12
    name = "second name"
  }
}
```

## Configuration layout

Configuration keys are mapped directly to flags.

Additionally, HCL block keys will be used as a hyphen-separated prefix when looking up flags.

## Example

The following HCL configuration file...

```hcl
debug = true

db {
    dsn = "root@/database"
    trace = true
}
```

Maps to the following flags:

```
--debug
--db-dsn=<string>
--db-trace
```
