package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vitaminmoo/memtools/hexpat/parser"
	"github.com/vitaminmoo/memtools/hexpat/codegen"
	"github.com/vitaminmoo/memtools/hexpat/resolve"
)

func main() {
	input := flag.String("i", "", "Input .hexpat file")
	output := flag.String("o", "", "Output .go file (default: stdout)")
	pkgName := flag.String("pkg", "", "Go package name (default: derived from input filename)")
	flag.Parse()

	if *input == "" {
		fmt.Fprintln(os.Stderr, "Usage: hexpatgen -i input.hexpat [-o output.go] [-pkg name]")
		os.Exit(1)
	}

	src, err := os.ReadFile(*input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading %s: %v\n", *input, err)
		os.Exit(1)
	}

	file, err := parser.Parse(string(src))
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse error: %v\n", err)
		os.Exit(1)
	}

	pkg, err := resolve.Resolve(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve error: %v\n", err)
		os.Exit(1)
	}

	name := *pkgName
	if name == "" {
		name = sanitizePkgName(filepath.Base(*input))
	}

	out, err := codegen.Generate(pkg, codegen.Options{PackageName: name})
	if err != nil {
		fmt.Fprintf(os.Stderr, "codegen error: %v\n", err)
		os.Exit(1)
	}

	if *output == "" {
		os.Stdout.Write(out)
	} else {
		if err := os.WriteFile(*output, out, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "error writing %s: %v\n", *output, err)
			os.Exit(1)
		}
	}
}

func sanitizePkgName(filename string) string {
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	name = strings.ToLower(name)
	var result []byte
	for _, c := range []byte(name) {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' {
			result = append(result, c)
		} else {
			result = append(result, '_')
		}
	}
	if len(result) > 0 && result[0] >= '0' && result[0] <= '9' {
		result = append([]byte{'_'}, result...)
	}
	if len(result) == 0 {
		return "generated"
	}
	return string(result)
}
