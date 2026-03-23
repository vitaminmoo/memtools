package main

import (
	goparser "go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vitaminmoo/memtools/hexpat/parser"
	"github.com/vitaminmoo/memtools/hexpat/codegen"
	"github.com/vitaminmoo/memtools/hexpat/resolve"
)

func TestPipeline(t *testing.T) {
	src := `
struct Header {
	u32 magic;
	u16 version;
};
`
	file, err := parser.Parse(src)
	require.NoError(t, err)

	pkg, err := resolve.Resolve(file)
	require.NoError(t, err)

	out, err := codegen.Generate(pkg, codegen.Options{PackageName: "test"})
	require.NoError(t, err)

	fset := token.NewFileSet()
	_, err = goparser.ParseFile(fset, "test.go", out, goparser.AllErrors)
	assert.NoError(t, err, "generated code does not parse:\n%s", string(out))
}

func TestSanitizePkgName(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"test.hexpat", "test"},
		{"my-file.hexpat", "my_file"},
		{"123start.hexpat", "_123start"},
		{"UPPER.hexpat", "upper"},
		{".hexpat", "generated"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, sanitizePkgName(tt.input), "sanitizePkgName(%q)", tt.input)
	}
}
