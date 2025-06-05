package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vlad/ast2llm-go/internal/types"
)

func TestParseFile(t *testing.T) {
	t.Parallel()

	p := New()

	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "valid file",
			input:    "package main; func foo() {}",
			expected: "main",
			wantErr:  false,
		},
		{
			name:    "invalid syntax",
			input:   "package main; func foo() {",
			wantErr: true,
		},
		{
			name:     "empty file",
			input:    "package main",
			expected: "main",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			file, err := p.Parse("test.go", []byte(tt.input))
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, file.Name.Name)
		})
	}
}

func TestExtractDeps(t *testing.T) {
	t.Parallel()

	p := New()

	tests := []struct {
		name     string
		input    string
		expected *types.FileInfo
	}{
		{
			name: "file with imports and functions",
			input: `package main

import (
	"fmt"
	"net/http"
)

func foo() {}
func bar() {}`,
			expected: &types.FileInfo{
				PackageName: "main",
				Imports:     []string{"fmt", "net/http"},
				Functions:   []string{"foo", "bar"},
			},
		},
		{
			name: "file with duplicate imports",
			input: `package test

import "fmt"
import "fmt"

func test() {}`,
			expected: &types.FileInfo{
				PackageName: "test",
				Imports:     []string{"fmt"},
				Functions:   []string{"test"},
			},
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			file, err := p.Parse("test.go", []byte(tt.input))
			assert.NoError(t, err)

			info := p.ExtractDeps(file)
			assert.Equal(t, tt.expected.PackageName, info.PackageName)
			assert.ElementsMatch(t, tt.expected.Imports, info.Imports)
			assert.ElementsMatch(t, tt.expected.Functions, info.Functions)
		})
	}
}
