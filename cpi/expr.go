package cpi

import (
	"github.com/khoakmp/kala/ast"
)

const (
	ScopeLocal = iota
	ScopeUpValue
	ScopeGlobal
	ScopeTable
)

type exprOption struct {
	resultSlot  int
	numRetValue int // -1 is variadic
}

func eOption(numRetValue int) exprOption {
	return exprOption{numRetValue: numRetValue, resultSlot: -1}
}

func getVarScope(fc *FunctionContext, varName string) (scope int) {
	index := fc.FindLocalVar(varName)
	if index > -1 {
		return ScopeLocal
	}
	fc = fc.Parent
	for fc != nil {
		if index := fc.FindLocalVar(varName); index > -1 {
			scope = ScopeUpValue
			return
		}
		fc = fc.Parent
	}
	return ScopeGlobal
}

func compileExprReduceMV(fc *FunctionContext, expr ast.Expr, slot *int, result *int) {
	if e, ok := expr.(*ast.IdentExpr); ok {
		idx := fc.FindLocalVar(e.Value)
		if idx > -1 {
			*result = idx
			return
		}
	}
	delta := compileExpr(fc, expr, *slot, eOption(1))
	*result = *slot
	*slot = *slot + delta
}

func compileExprReduceLKMV(fc *FunctionContext, expr ast.Expr, slot *int, result *int) {
	if e, ok := expr.(*ast.StringExpr); ok {
		kidx := fc.Consts.IndexOf(kString(e.Value))
		*result = opRkAsk(kidx)
		return
	}

	if e, ok := expr.(*ast.NumberExpr); ok {
		kidx := fc.Consts.IndexOf(kNumber(e.Value))
		*result = opRkAsk(kidx)
		return
	}

	if e, ok := expr.(*ast.IdentExpr); ok {
		idx := fc.FindLocalVar(e.Value)
		if idx > -1 {
			*result = idx
			return
		}
	}
	slotUsed := compileExpr(fc, expr, *slot, eOption(1))
	*result = *slot
	*slot = *slot + slotUsed
}

func compileExpr(fc *FunctionContext, expr ast.Expr, slot int, opt exprOption) int {
	rslot := slot
	if opt.resultSlot != -1 {
		rslot = opt.resultSlot
	}
	delta := 1
	if rslot < slot || opt.numRetValue == 0 {
		delta = 0
	}

	switch e := expr.(type) {
	case *ast.StringExpr:
		s := fc.Consts.IndexOf(kString(e.Value))
		fc.AddInst(opCreateABx(OP_LOADK, rslot, s))
		return delta
	case *ast.NumberExpr:
		s := fc.Consts.IndexOf(kNumber(e.Value))
		fc.AddInst(opCreateABx(OP_LOADK, rslot, s))
		return delta
	case *ast.NilExpr:
		fc.Inst.AddNil(rslot, rslot)
		return delta
	case *ast.TrueExpr:
		fc.AddInst(opCreateABC(OP_LOADBOOL, rslot, 1, 0))
		return delta
	case *ast.FalseExpr:
		fc.AddInst(opCreateABC(OP_LOADBOOL, rslot, 0, 0))
		return delta
	case *ast.IdentExpr:
		scope := getVarScope(fc, e.Value)
		switch scope {
		case ScopeLocal:
			b := fc.FindLocalVar(e.Value)
			fc.AddInst(opCreateABC(OP_MOVE, rslot, b, 0))
		case ScopeUpValue:
			b := fc.Upvalues.GetUnique(e.Value)
			fc.AddInst(opCreateABC(OP_GETUPVAL, rslot, b, 0))
		case ScopeGlobal:
			b := fc.Consts.IndexOf(KString(e.Value))
			fc.AddInst(opCreateABx(OP_GETGLOBAL, rslot, b))
		}
		return delta
	case *ast.FieldGetExpr:
		var oslot int
		compileExprReduceMV(fc, e.Object, &slot, &oslot)

		if k, ok := e.Key.(*ast.StringExpr); ok {
			c := fc.Consts.IndexOf(KString(k.Value))
			fc.AddInst(opCreateABC(OP_GETTABLEKS, rslot, oslot, c))
			return delta
		}
		compileExpr(fc, e.Key, slot, eOption(1))
		fc.AddInst(opCreateABC(OP_GETTABLE, rslot, oslot, slot))
		return delta
	case *ast.ArithmeticOpExpr:
		return compileArithmeticOpExpr(fc, e, slot, opt)
	case *ast.FuncCallExpr:
		return compileFuncCallExpr(fc, e, slot, opt)

	case *ast.FunctionExpr:
		childCtx := NewFunctionContext(fc, len(e.Params), e.HasVArg)
		compileFuncExpr(childCtx, e)
		bx := len(fc.Proto.FuncProtos)
		fc.Proto.AddChildProto(childCtx.Proto)

		fc.AddInst(opCreateABx(OP_CLOSURE, rslot, bx))

		for _, v := range childCtx.Upvalues.List() {
			if idx, block := fc.FindLocaVarAndBlock(v); idx > -1 {
				block.NeedClose = true
				fc.AddInst(opCreateABC(OP_MOVE, 0, idx, 0))
			} else {
				idx := fc.Upvalues.GetUnique(v)
				fc.AddInst(opCreateABC(OP_GETUPVAL, 0, idx, 0))
			}
		}
		return delta
	case *ast.RelationalOpExpr:
		return compileRelationalExpr(fc, e, slot, opt)
	case *ast.DictExpr:
		return compileDictExpr(fc, e, slot, opt)
	case *ast.ListExpr:
		return compileListExprV2(fc, e, slot, opt)
	case *ast.LenExpr:
		return compileLenExpr(fc, e, slot, opt)
	case *ast.ConcatStrExpr:
		var b, c int
		compileExprReduceMV(fc, e.Lhs, &slot, &b)
		compileExprReduceMV(fc, e.Rhs, &slot, &c)
		fc.AddInst(opCreateABC(OP_CONCAT, rslot, b, c))
		return delta
	case *ast.UnaryOpMinusExpr:
		var b int
		compileExprReduceMV(fc, e.Expr, &slot, &b)
		fc.AddInst(opCreateABC(OP_UNM, rslot, b, 0))
		return delta
	case *ast.UnaryOpNotExpr:
		var b int
		compileExprReduceMV(fc, e.Expr, &slot, &b)
		fc.AddInst(opCreateABC(OP_NOT, rslot, b, 0))
		return delta
	case *ast.LogicalOpExpr:
		return compileLogicalOpExpr(fc, e, slot, opt)
	}

	return 0
}
func compileLogicalOpExpr(fc *FunctionContext, expr *ast.LogicalOpExpr, slot int, opt exprOption) int {
	rslot := slot
	if opt.resultSlot != -1 {
		rslot = opt.resultSlot
	}
	delta := 1
	if rslot < slot {
		delta = 0
	}
	trueLabel := fc.NewLabel()
	falseLabel := fc.NewLabel()

	if expr.Operator == ast.OpAnd {
		nextCondLabel := fc.NewLabel()
		compileBranchCond(fc, expr.Lhs, slot, nextCondLabel, falseLabel, nextCondLabel)
		fc.MarkLabel(nextCondLabel, fc.Inst.LastIndex())
		compileBranchCond(fc, expr.Rhs, slot, trueLabel, falseLabel, trueLabel)

		fc.MarkLabel(trueLabel, fc.Inst.LastIndex())
		fc.AddInst(opCreateABC(OP_LOADBOOL, rslot, 1, 1))

		fc.MarkLabel(falseLabel, fc.Inst.LastIndex())
		fc.AddInst(opCreateABC(OP_LOADBOOL, rslot, 0, 0))
		return delta
	}
	// OR
	nextCondLabel := fc.NewLabel()
	compileBranchCond(fc, expr.Lhs, slot, trueLabel, nextCondLabel, nextCondLabel)
	fc.MarkLabel(nextCondLabel, fc.Inst.LastIndex())
	compileBranchCond(fc, expr.Rhs, slot, trueLabel, falseLabel, falseLabel)

	fc.MarkLabel(falseLabel, fc.Inst.LastIndex())
	fc.AddInst(opCreateABC(OP_LOADBOOL, rslot, 0, 1))

	fc.MarkLabel(trueLabel, fc.Inst.LastIndex())
	fc.AddInst(opCreateABC(OP_LOADBOOL, rslot, 1, 0))
	return delta
}
func compileArithmeticOpExpr(fc *FunctionContext, expr *ast.ArithmeticOpExpr, slot int, opt exprOption) int {
	var a, b, c = slot, 0, 0
	delta := 1
	if opt.resultSlot != -1 {
		a = opt.resultSlot
	}
	if a < slot || opt.numRetValue == 0 {
		delta = 0
	}
	compileExprReduceLKMV(fc, expr.Lhs, &slot, &b)
	compileExprReduceLKMV(fc, expr.Rhs, &slot, &c)
	var opcode int
	switch expr.Operator {
	case ast.OpAdd:
		opcode = OP_ADD
	case ast.OpSubtract:
		opcode = OP_SUB
	case ast.OpMul:
		opcode = OP_MUL
	case ast.OpDiv:
		opcode = OP_DIV
	case ast.OpMod:
		opcode = OP_MOD
	}
	fc.AddInst(opCreateABC(opcode, a, b, c))
	return delta
}

func compileFuncCallExpr(fc *FunctionContext, expr *ast.FuncCallExpr, slot int, opt exprOption) int {
	compileExpr(fc, expr.Func, slot, eOption(1)) // always incr 1
	narg := len(expr.Args)

	if narg == 0 {
		fc.AddInst(opCreateABC(OP_CALL, slot, 1, opt.numRetValue+1))
		if opt.numRetValue < 0 {
			return 0
		}
		return opt.numRetValue
	}

	start := slot + 1

	for i := range narg - 1 {
		start += compileExpr(fc, expr.Args[i], start, eOption(1))
	}
	delta := compileExpr(fc, expr.Args[narg-1], start, eOption(-1))
	b := narg + delta
	if delta == 0 {
		b = 0
	}

	fc.AddInst(opCreateABC(OP_CALL, slot, b, opt.numRetValue+1))
	if opt.numRetValue < 0 {
		return 0
	}
	return opt.numRetValue
}

func compileFuncExpr(fc *FunctionContext, expr *ast.FunctionExpr) {
	fc.Proto.NumParams = len(expr.Params)
	fc.Proto.HasVarg = expr.HasVArg
	for _, v := range expr.Params {
		fc.AddLocalVar(v)
	}
	if expr.HasVArg {
		fc.AddLocalVar("arg")
	}
	compileChunk(fc, expr.Block)
	fc.AddInst(opCreateABC(OP_RETURN, 0, 1, 0))
	fc.Proto.Consts = fc.Consts
	fc.Proto.NumUpvalues = fc.Upvalues.Len()
	fc.Proto.InstList = fc.Inst

	stringConsts := make([]string, fc.Consts.Len())
	for i, s := range fc.Consts.data {
		if s, ok := s.(KString); ok {
			stringConsts[i] = string(s)
		}
	}
	fc.Proto.StringConsts = stringConsts
	patchCode(fc)
}

func compileRelationalOpExprAux(fc *FunctionContext, expr *ast.RelationalOpExpr, slot int, a int, jumpLabel int) {
	var b, c int
	compileExprReduceLKMV(fc, expr.Lhs, &slot, &b)
	compileExprReduceLKMV(fc, expr.Rhs, &slot, &c)
	switch expr.Operator {
	case ast.OpLt:
		fc.AddInst(opCreateABC(OP_LT, a, b, c))
	case ast.OpGt:
		fc.AddInst(opCreateABC(OP_LT, a, c, b))
	case ast.OpLe:
		fc.AddInst(opCreateABC(OP_LE, a, b, c))
	case ast.OpGe:
		fc.AddInst(opCreateABC(OP_LE, a, c, b))
	case ast.OpEqual:
		fc.AddInst(opCreateABC(OP_EQ, a, b, c))
	case ast.OpNotEqual:
		fc.AddInst(opCreateABC(OP_EQ, 1^a, b, c))
	}
	fc.AddInst(opCreateASbx(OP_JMP, 0, jumpLabel))
}

func compileRelationalExpr(fc *FunctionContext, expr *ast.RelationalOpExpr, slot int, opt exprOption) int {
	rslot := slot
	if opt.resultSlot != -1 {
		rslot = opt.resultSlot
	}
	delta := 1
	if rslot < slot || opt.numRetValue == 0 {
		delta = 0
	}

	trueLabel := fc.NewLabel()
	falseLabel := fc.NewLabel()
	compileRelationalOpExprAux(fc, expr, slot, 0, falseLabel)
	fc.MarkLabel(trueLabel, fc.Inst.LastIndex())
	fc.AddInst(opCreateABC(OP_LOADBOOL, slot, 1, 1))

	fc.MarkLabel(falseLabel, fc.Inst.LastIndex())
	fc.AddInst(opCreateABC(OP_LOADBOOL, slot, 0, 0))
	return delta
}

func compileBranchCond(fc *FunctionContext, expr ast.Expr, slot int, thenLabel, elseLabel, nextLabel int) {
	switch e := expr.(type) {
	case *ast.FalseExpr, *ast.NilExpr:
		if nextLabel == thenLabel {
			fc.AddInst(opCreateASbx(OP_JMP, 0, elseLabel))
		}
		// else nextLabel is elseLabel, so not need to generate code
		return
	case *ast.TrueExpr:
		if nextLabel == elseLabel {
			fc.AddInst(opCreateASbx(OP_JMP, 0, thenLabel))
		}
		return
	/* case *ast.StringExpr, *ast.NumberExpr:
	goto CAL */
	case *ast.RelationalOpExpr:
		if nextLabel == thenLabel {
			compileRelationalOpExprAux(fc, e, slot, 0, elseLabel)
		} else {
			// case: nextLabel == elseLabel
			compileRelationalOpExprAux(fc, e, slot, 1, thenLabel)
		}

		return
	case *ast.UnaryOpNotExpr:
		compileBranchCond(fc, e.Expr, slot, elseLabel, thenLabel, nextLabel)
		return
	case *ast.LogicalOpExpr:
		if e.Operator == ast.OpAnd {
			nextCondLabel := fc.NewLabel()
			compileBranchCond(fc, e.Lhs, slot, nextCondLabel, elseLabel, nextCondLabel)
			fc.MarkLabel(nextCondLabel, fc.Inst.LastIndex())
			compileBranchCond(fc, e.Rhs, slot, thenLabel, elseLabel, nextLabel)
		} else {
			// Operator OR
			nextCondLabel := fc.NewLabel()
			compileBranchCond(fc, e.Lhs, slot, thenLabel, nextCondLabel, nextCondLabel)
			fc.MarkLabel(nextCondLabel, fc.Inst.LastIndex())
			compileBranchCond(fc, e.Rhs, slot, thenLabel, elseLabel, nextLabel)
		}
		return

	}

	//CAL:
	compileExpr(fc, expr, slot, eOption(1))
	if nextLabel == thenLabel {
		fc.AddInst(opCreateABC(OP_TEST, slot, 0, 0))
		fc.AddInst(opCreateASbx(OP_JMP, 0, elseLabel))
	}
	// case nextLabel == elseLabel:
	fc.AddInst(opCreateABC(OP_TEST, slot, 0, 1))
	fc.AddInst(opCreateASbx(OP_JMP, 0, thenLabel))

}

func compileDictExpr(fc *FunctionContext, expr *ast.DictExpr, slot int, opt exprOption) int {
	l := max(1, len(expr.Entries))

	rslot := slot
	delta := 1
	if opt.resultSlot != -1 {
		rslot = opt.resultSlot
	}

	if rslot < slot || opt.numRetValue == 0 {
		delta = 0
	}
	fc.AddInst(opCreateABC(OP_NEWTABLE, rslot, 0, l))
	eslot := slot + 1
	for _, entry := range expr.Entries {
		kidx := fc.Consts.IndexOf(KString(entry.Key))
		compileExpr(fc, entry.Value, eslot, eOption(1))
		fc.AddInst(opCreateABC(OP_SETTABLEKS, rslot, kidx, eslot))
	}
	return delta
}

func compileListExprV2(fc *FunctionContext, expr *ast.ListExpr, slot int, opt exprOption) int {
	l := max(1, len(expr.Elements))

	a := slot
	delta := 1
	if opt.resultSlot != -1 {
		a = opt.resultSlot
	}
	if a < slot || opt.numRetValue == 0 {
		delta = 0
	}

	fc.AddInst(opCreateABC(OP_NEWTABLE, a, l, 0))
	if len(expr.Elements) == 0 {
		return delta
	}
	slot++
	for _, e := range expr.Elements {
		slot += compileExpr(fc, e, slot, eOption(1))
	}
	// not use c
	fc.AddInst(opCreateABC(OP_SETLIST, a, l, 0))
	return delta
}

func compileLenExpr(fc *FunctionContext, expr *ast.LenExpr, slot int, opt exprOption) int {
	var oslot int
	rslot := slot
	delta := 1
	if opt.resultSlot != -1 {
		rslot = opt.resultSlot
	}
	if rslot < slot || opt.numRetValue == 0 {
		delta = 0
	}
	compileExprReduceMV(fc, expr.Object, &slot, &oslot)
	// R(A) := length of R(B)
	fc.AddInst(opCreateABC(OP_LEN, rslot, oslot, 0))
	return delta
}
