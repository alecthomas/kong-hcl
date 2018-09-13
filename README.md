# A Kong configuration loader for HCL [![](https://godoc.org/github.com/alecthomas/kong-hcl?status.svg)](http://godoc.org/github.com/alecthomas/kong-hcl) [![CircleCI](https://img.shields.io/circleci/project/github/alecthomas/kong-hcl.svg)](https://circleci.com/gh/alecthomas/kong-hcl)

Use it like so:

```go
var cli struct {
    Config kong.ConfigFlag `help:"Load configuration."`
}
parser, err := kong.New(&cli, kong.Configuration(konghcl.Loader, "/etc/myapp/config.hcl", "~/.myapp.hcl))
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