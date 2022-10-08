package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"log"
	"os"
	"sort"

	"github.com/daichitakahashi/gocmd/internal"
)

var (
	dst, pkg, varName string
)

func init() {
	flag.StringVar(&dst, "dst", "", "out put file path")
	flag.StringVar(&pkg, "pkg", "", "out put package name")
	flag.StringVar(&varName, "var", "", "out put variable name")
}

func main() {
	flag.Parse()
	if dst == "" {
		log.Fatal("output file path not specified")
	}
	if pkg == "" {
		log.Fatal("output package name not specified")
	}
	if varName == "" {
		log.Fatal("output variable name not specified")
	}

	_, err := internal.FetchOnce()
	if err != nil {
		log.Fatal(err)
	}

	type versionStable struct {
		version string
		stable  bool
	}
	var list []versionStable
	internal.Versions(func(versions map[string]bool) {
		list = make([]versionStable, 0, len(versions))
		for v, s := range versions {
			list = append(list, versionStable{
				version: v,
				stable:  s,
			})
		}
	})
	sort.Slice(list, func(i, j int) bool {
		return list[i].version < list[j].version
	})

	buf := bytes.NewBuffer(nil)
	_, _ = fmt.Fprintf(buf, `// Code generated by genvers. DO NOT EDIT.
package %s

var %s = map[string]bool{
`, pkg, varName)

	for _, item := range list {
		_, _ = fmt.Fprintf(buf, `%#v: %t,
`, item.version, item.stable)
	}

	buf.WriteByte('}')

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatal(err)
	}

	err = os.WriteFile(dst, formatted, 0644)
	if err != nil {
		log.Fatal(err)
	}
}
