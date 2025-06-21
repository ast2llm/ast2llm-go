package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewFileInfo(t *testing.T) {
	fi := NewFileInfo()
	assert.NotNil(t, fi)
	assert.Empty(t, fi.PackageName)
	assert.NotNil(t, fi.Imports)
	assert.NotNil(t, fi.Functions)
	assert.NotNil(t, fi.Structs)
	assert.NotNil(t, fi.Interfaces)
	assert.NotNil(t, fi.GlobalVars)
	assert.NotNil(t, fi.UsedImportedStructs)
	assert.NotNil(t, fi.UsedImportedFunctions)
	assert.NotNil(t, fi.UsedImportedGlobalVars)
}

func TestNewStructField(t *testing.T) {
	f := NewStructField()
	assert.NotNil(t, f)
	assert.Empty(t, f.Name)
	assert.Empty(t, f.Type)
}

func TestNewStructMethod(t *testing.T) {
	m := NewStructMethod()
	assert.NotNil(t, m)
	assert.Empty(t, m.Name)
	assert.Empty(t, m.Comment)
	assert.NotNil(t, m.Parameters)
	assert.NotNil(t, m.ReturnTypes)
	assert.Empty(t, m.Parameters)
	assert.Empty(t, m.ReturnTypes)
}

func TestNewStructInfo(t *testing.T) {
	si := NewStructInfo()
	assert.NotNil(t, si)
	assert.Empty(t, si.Name)
	assert.Empty(t, si.Comment)
	assert.NotNil(t, si.Fields)
	assert.NotNil(t, si.Methods)
	assert.Empty(t, si.Fields)
	assert.Empty(t, si.Methods)
}

func TestNewNode(t *testing.T) {
	n := NewNode()
	assert.NotNil(t, n)
	assert.Empty(t, n.PkgPath)
	assert.NotNil(t, n.Functions)
	assert.NotNil(t, n.DependsOn)
	assert.NotNil(t, n.Files)
	assert.Empty(t, n.Functions)
	assert.Empty(t, n.DependsOn)
	assert.Empty(t, n.Files)
}

func TestNewDependencyGraph(t *testing.T) {
	dg := NewDependencyGraph()
	assert.NotNil(t, dg)
	assert.NotNil(t, dg.Nodes)
	assert.Empty(t, dg.Nodes)
}

func TestNewInterfaceMethod(t *testing.T) {
	im := NewInterfaceMethod()
	assert.NotNil(t, im)
	assert.Empty(t, im.Name)
	assert.Empty(t, im.Comment)
	assert.NotNil(t, im.Parameters)
	assert.NotNil(t, im.ReturnTypes)
	assert.Empty(t, im.Parameters)
	assert.Empty(t, im.ReturnTypes)
}

func TestNewInterfaceInfo(t *testing.T) {
	ii := NewInterfaceInfo()
	assert.NotNil(t, ii)
	assert.Empty(t, ii.Name)
	assert.Empty(t, ii.Comment)
	assert.NotNil(t, ii.Methods)
	assert.NotNil(t, ii.Embeddeds)
	assert.Empty(t, ii.Methods)
	assert.Empty(t, ii.Embeddeds)
}

func TestNewGlobalVarInfo(t *testing.T) {
	gv := NewGlobalVarInfo()
	assert.NotNil(t, gv)
	assert.Empty(t, gv.Name)
	assert.Empty(t, gv.Comment)
	assert.Empty(t, gv.Type)
	assert.Empty(t, gv.Value)
	assert.False(t, gv.IsConst)
}

func TestNewFunctionInfo(t *testing.T) {
	fn := NewFunctionInfo()
	assert.NotNil(t, fn)
	assert.Empty(t, fn.Name)
	assert.Empty(t, fn.Comment)
	assert.NotNil(t, fn.Params)
	assert.NotNil(t, fn.Returns)
	assert.Empty(t, fn.Params)
	assert.Empty(t, fn.Returns)
}
