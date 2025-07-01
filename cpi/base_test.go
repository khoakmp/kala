package cpi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEx(t *testing.T) {

}
func TestFunctionContext(t *testing.T) {
	fc := NewFunctionContext(nil, 0, false)
	fc.AddLocalVar("a")
	fc.AddLocalVar("b")
	sa := fc.FindLocalVar("a")
	assert.Equal(t, 0, sa)
	//assert.Equal(t, 1, fc.FindLocalVar("b"))
	fc.EnterBlock(NoBreakLabel)
	fc.AddLocalVar("c")
	assert.Equal(t, 2, fc.FindLocalVar("c"))
	assert.Equal(t, 1, fc.FindLocalVar("b"))
	fc.AddLocalVar("a")
	assert.Equal(t, 3, fc.FindLocalVar("a"))
	assert.Equal(t, 4, fc.StackTop())
	fc.LeaveBlock(true)
	assert.Equal(t, 2, fc.StackTop())
	assert.Equal(t, ScopeLocal, getVarScope(fc, "b"))

	assert.Equal(t, 0, fc.Consts.IndexOf(KString("kmp")))

}
