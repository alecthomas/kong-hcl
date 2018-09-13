package konghcl

import (
	"fmt"
	"strings"

	"github.com/alecthomas/participle"
)

// Parser for HCL.
var Parser = participle.MustBuild(&Config{})

func findEntries(entries []*Entry, path []string) *Value {
	if len(path) == 0 {
		return nil
	}
	for _, entry := range entries {
		if entry.Key == path[0] {
			return entry.Find(path[1:])
		}
		prefix := entry.Key + "-"
		if strings.HasPrefix(path[0], prefix) {
			path = append([]string{strings.TrimPrefix(path[0], prefix)}, path[1:]...)
			return entry.Find(path)
		}
	}
	return nil
}

// Config is the root configuration structure.
type Config struct {
	Entries []*Entry `{ @@ }`
}

func (c *Config) Find(path []string) *Value { // nolint: golint
	return findEntries(c.Entries, path)
}

// A Block is a group of HCL entries.
type Block struct {
	Parameters []*Value `{ @@ }`
	Entries    []*Entry `"{" { @@ } "}"`
}

func (b *Block) Find(path []string) *Value { // nolint: golint
	return findEntries(b.Entries, path)
}

// An Entry in a HCL block.
type Entry struct {
	Key   string `@( Ident { "-" Ident } )`
	Value *Value `( "=" @@`
	Block *Block `| @@ )`
}

func (e *Entry) Find(path []string) *Value { // nolint: golint
	if e.Block != nil {
		return e.Block.Find(path)
	}
	if len(path) == 0 {
		return e.Value
	}
	return nil
}

// A Value for a key in HCL.
type Value struct {
	Boolean    *Bool    `  @("true"|"false")`
	Identifier *string  `| @( Ident { "-" Ident } { "." Ident { "-" Ident } } )`
	Str        *string  `| @(String|Char|RawString)`
	Number     *float64 `| @(Float|Int)`
	Array      []*Value `| "[" { @@ [ "," ] } "]"`
}

func (v *Value) String() string {
	switch {
	case v.Boolean != nil:
		return fmt.Sprintf("%v", *v.Boolean)
	case v.Identifier != nil:
		return *v.Identifier
	case v.Str != nil:
		return *v.Str
	case v.Number != nil:
		return fmt.Sprintf("%v", *v.Number)
	case v.Array != nil:
		out := []string{}
		for _, v := range v.Array {
			out = append(out, v.String())
		}
		return strings.Join(out, ",")
	}
	panic("??")
}

func (v *Value) GoString() string { // nolint: golint
	if v == nil {
		return "nil"
	}
	switch {
	case v.Boolean != nil:
		return fmt.Sprintf("%v", *v.Boolean)
	case v.Identifier != nil:
		return fmt.Sprintf("`%s`", *v.Identifier)
	case v.Str != nil:
		return fmt.Sprintf("%q", *v.Str)
	case v.Number != nil:
		return fmt.Sprintf("%v", *v.Number)
	case v.Array != nil:
		out := []string{}
		for _, v := range v.Array {
			out = append(out, v.GoString())
		}
		return fmt.Sprintf("[]*Value{ %s }", strings.Join(out, ", "))
	}
	panic("??")
}

// A Bool value.
type Bool bool

func (b *Bool) Capture(v []string) error { *b = v[0] == "true"; return nil } // nolint: golint
