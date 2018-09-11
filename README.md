# A Kong configuration loader for HCL [![](https://godoc.org/github.com/alecthomas/kong-hcl?status.svg)](http://godoc.org/github.com/alecthomas/kong-hcl) [![CircleCI](https://img.shields.io/circleci/project/github/alecthomas/kong-hcl.svg)](https://circleci.com/gh/alecthomas/kong-hcl)

Use it like so:

```go
parser, err := kong.New(&cli, kong.Configuration(konghcl.Loader, "/etc/myapp/config.hcl", "~/.myapp.hcl))
```