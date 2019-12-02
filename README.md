# A Kong configuration loader for HCL [![](https://godoc.org/github.com/alecthomas/kong-hcl?status.svg)](http://godoc.org/github.com/alecthomas/kong-hcl) [![CircleCI](https://img.shields.io/circleci/project/github/alecthomas/kong-hcl.svg)](https://circleci.com/gh/alecthomas/kong-hcl)

Use it like so:

```go
var cli struct {
    Config kong.ConfigFlag `help:"Load configuration."`
}
parser, err := kong.New(&cli, kong.Configuration(konghcl.Loader, "/etc/myapp/config.hcl", "~/.myapp.hcl"))
```

## Mapping HCL fragments to a struct

More complex structures can be loaded directly into flag values by implementing the
`kong.MapperValue` interface, and calling `konghcl.DecodeValue`. 

The value can either be a HCL(/JSON) fragment, or a path to a HCL file that will be loaded. Both
can be specified on the command-line or config file.

eg.

```go
type NestedConfig struct {
	Size int
	Name string
}

type ComplexConfig struct {
	Key bool
	Nested map[string]NestedConfig
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
