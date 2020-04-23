package konghcl

import (
	"fmt"
	"sort"
	"strings"

	"github.com/alecthomas/kong"
)

var (
	// DumpIgnoreFlags specifies a set of flags that should not be dumped.
	DumpIgnoreFlags = map[string]bool{
		"help": true, "version": true, "dump-config": true, "env": true, "validate-config": true,
	}
)

// DumpConfig can be added as a flag to dump HCL configuration.
type DumpConfig bool

func (f DumpConfig) BeforeApply(app *kong.Kong) error { // nolint: golint
	groups := map[string][]*kong.Flag{}
	standalone := []*kong.Flag{}
	for _, flags := range app.Model.AllFlags(true) {
		for _, flag := range flags {
			if DumpIgnoreFlags[flag.Name] {
				continue
			}
			parts := strings.SplitN(flag.Name, "-", 2)
			if len(parts) == 1 {
				standalone = append(standalone, flag)
			} else {
				groups[parts[0]] = append(groups[parts[0]], flag)
			}
		}
	}

	// Write non-grouped flags out at the top.
	for key, flags := range groups {
		if len(flags) == 1 {
			standalone = append(standalone, flags...)
			delete(groups, key)
		}
	}

	// Alphabetical ordering.
	sort.Slice(standalone, func(i, j int) bool {
		return standalone[i].Name < standalone[j].Name
	})
	for _, flag := range standalone {
		formatFlag("", flag, false)
		fmt.Println()
	}
	delete(groups, "")

	// Alphabetically order the groups.
	keys := []string{}
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for block := range groups {
		flags := groups[block]
		if len(flags) == 1 {
			formatFlag("", flags[0], false)
			fmt.Println()
			continue
		}
		fmt.Printf("%s {\n", block)
		for i, flag := range flags {
			if i != 0 {
				fmt.Println()
			}
			formatFlag("  ", flag, true)
		}
		fmt.Printf("}\n\n")
	}
	app.Exit(0)
	return nil
}

func formatFlag(indent string, flag *kong.Flag, grouped bool) {
	fmt.Printf("%s// %s\n", indent, flag.Help)
	fmt.Print(indent)
	if grouped {
		parts := strings.SplitN(flag.Name, "-", 2)
		fmt.Printf("%s = ", parts[1])
	} else {
		fmt.Printf("%s = ", flag.Name)
	}
	switch {
	case flag.IsSlice():
		fmt.Println("[ ... ]")
	case flag.IsMap():
		fmt.Println("{ ... }")
	default:
		fmt.Println(flag.FormatPlaceHolder())
	}
}
