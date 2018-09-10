# A Kong configuration loader for HCL [![](https://godoc.org/github.com/alecthomas/konghcl?status.svg)](http://godoc.org/github.com/alecthomas/konghcl) [![CircleCI](https://img.shields.io/circleci/project/github/alecthomas/konghcl.svg)](https://circleci.com/gh/alecthomas/konghcl)

Use it like so:

```go
parser, err := kong.New(&cli, kong.Configuration(konghcl.Loader, "/etc/myapp/config.hcl", "~/.myapp.hcl))
```