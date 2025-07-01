package cpi

import (
	"fmt"

	"github.com/khoakmp/kala/ast"
)

func compileStmt(fc *FunctionContext, stmt ast.Stmt) {
	switch stmt := stmt.(type) {
	case *ast.AssignStmt:
		compileAssignStmt(fc, stmt)
	case *ast.IfStmt:
		compileIfStmt(fc, stmt)
	case *ast.WhileStmt:
		compileWhileStmt(fc, stmt)
	case *ast.ForNumberStmt:
		compileForNumberStmt(fc, stmt)
	case *ast.BreakStmt:
		compileBreakStmt(fc, stmt)
	case *ast.ReturnStmt:
		compileReturnStmt(fc, stmt)
	case *ast.VarDefStmt:
		compileVarDefStmt(fc, stmt)
	case *ast.FuncDefStmt:
		compileFuncDefStmt(fc, stmt)
	case *ast.FuncCallStmt:
		compileFuncCallExpr(fc, stmt.Expr, fc.StackTop(), eOption(0))
	case *ast.ListAppendStmt:
		compileListAppendStmt(fc, stmt)
	case *ast.ForRangeStmt:
		compileForRangeStmt(fc, stmt)
	}
}

type assignLeft struct {
	left, key, scope int
}

func compileAssignStmt(fc *FunctionContext, stmt *ast.AssignStmt) {
	/* if len(stmt.Lhs) == 1 && len(stmt.Rhs) == 1 {
		if _, ok := stmt.Rhs[0].(*ast.FuncCallExpr); !ok {
			delta, ags := compileAssginLeftList(fc, stmt)
			slot := fc.StackTop() + delta
			opt := eOption(1)
			if ags[0].scope == ScopeLocal {
				opt.resultSlot = ags[0].left
			}

			compileExpr(fc, stmt.Rhs[0], slot, opt)
			switch ags[0].scope {
			case ScopeLocal:

			case ScopeUpValue:
				fc.Inst.Add(opCreateABx(OP_SETUPVAL, slot, ags[0].left))
			case ScopeGlobal:
				fc.Inst.Add(opCreateABx(OP_SETGLOBAL, slot, ags[0].left))
			case ScopeTable:
				fc.Inst.Add(opCreateABC(OP_SETTABLE, ags[0].left, ags[0].key, slot))
			}
			return
		}
	} */
	delta, ags := compileAssginLeftList(fc, stmt)
	slot := fc.StackTop() + delta
	lsize, rsize := len(stmt.Lhs), len(stmt.Rhs)

	compileRight := func(start, end int, slot, num int) {
		for i := start; i <= end; i++ {
			slot += compileExpr(fc, stmt.Rhs[i], slot, eOption(num))
		}
	}

	assignFn := func(start, end, slot int) {
		for i := end; i >= start; i-- {
			switch ags[i].scope {
			case ScopeLocal:
				fc.Inst.Add(opCreateABC(OP_MOVE, ags[i].left, slot, 0))
			case ScopeUpValue:
				fc.Inst.Add(opCreateABx(OP_SETUPVAL, slot, ags[i].left))
			case ScopeGlobal:
				fc.Inst.Add(opCreateABx(OP_SETGLOBAL, slot, ags[i].left))
			case ScopeTable:
				fc.Inst.Add(opCreateABC(OP_SETTABLE, ags[i].left, ags[i].key, slot))
			}
			slot--
		}
	}

	if lsize == rsize {
		compileRight(0, rsize-1, slot, 1)
		assignFn(0, lsize-1, slot+lsize-1)
		return
	}

	if lsize < rsize {
		compileRight(0, len(stmt.Lhs)-1, slot, 1)
		markSlot := slot + len(stmt.Lhs) - 1
		compileRight(len(stmt.Lhs), len(stmt.Rhs)-1, markSlot+1, 0)
		assignFn(len(stmt.Lhs)-1, 0, markSlot)
		return
	}
	// len stmt.Lhs > len(stmt.Rhs)
	if rsize == 0 {
		// raise error
		panic("")
	}
	if rsize > 1 {
		compileRight(0, rsize-2, slot, 1)
		slot += rsize - 1
	}

	remain := lsize - rsize + 1
	delta = compileExpr(fc, stmt.Rhs[rsize-1], slot, eOption(remain))
	if delta < remain {
		panic("")
	}
	slot += delta
	assignFn(0, lsize-1, slot-1)
}

func compileAssginLeftList(fc *FunctionContext, stmt *ast.AssignStmt) (slotUsed int, ags []assignLeft) {
	slot := fc.StackTop()
	ags = make([]assignLeft, len(stmt.Lhs))
	for i, e := range stmt.Lhs {
		switch e := e.(type) {
		case *ast.IdentExpr:
			ags[i].scope = getVarScope(fc, e.Value)
			switch ags[i].scope {
			case ScopeGlobal:
				ags[i].left = fc.Consts.IndexOf(KString(e.Value))
			case ScopeLocal:
				ags[i].left = fc.FindLocalVar(e.Value)
				//fmt.Println("left:", ags[i].left)
				//fmt.Println("var", e.Value, "at slot:", ags[i].left)
			case ScopeUpValue:
				ags[i].left = fc.Upvalues.GetUnique(e.Value)
			}
		case *ast.FieldGetExpr:
			ags[i].scope = ScopeTable

			compileExprReduceLKMV(fc, e.Object, &slot, &ags[i].left)
			compileExprReduceLKMV(fc, e.Key, &slot, &ags[i].key)
		}

	}
	slotUsed = slot - fc.StackTop()
	return
}

func compileBlock(fc *FunctionContext, chunk []ast.Stmt) {
	fc.EnterBlock(NoBreakLabel)
	for _, stmt := range chunk {
		compileStmt(fc, stmt)
	}
	fc.LeaveBlock(true)
}

func compileIfStmt(fc *FunctionContext, stmt *ast.IfStmt) {
	endLabel := fc.NewLabel()
	thenLabel := fc.NewLabel()
	elseLabel := fc.NewLabel()

	compileBranchCond(fc, stmt.CondExpr, fc.StackTop(), thenLabel, elseLabel, thenLabel)
	fc.MarkLabel(thenLabel, fc.Inst.LastIndex())
	compileBlock(fc, stmt.ThenChunk)

	if len(stmt.ElseChunk) > 0 {
		fc.AddInst(opCreateASbx(OP_JMP, 0, endLabel))
	}
	fc.MarkLabel(elseLabel, fc.Inst.LastIndex())
	if len(stmt.ElseChunk) > 0 {
		compileBlock(fc, stmt.ElseChunk)
	}
	fc.MarkLabel(endLabel, fc.Inst.LastIndex())
}

func compileWhileStmt(fc *FunctionContext, stmt *ast.WhileStmt) {
	endLabel := fc.NewLabel()
	condLabel := fc.NewLabel()
	doLabel := fc.NewLabel()

	fc.EnterBlock(endLabel)
	fc.MarkLabel(condLabel, fc.Inst.LastIndex())
	compileBranchCond(fc, stmt.CondExpr, fc.StackTop(), doLabel, endLabel, doLabel)
	fc.MarkLabel(doLabel, fc.Inst.LastIndex())
	compileChunk(fc, stmt.Chunk)
	// manually close Upvalues
	// at runtime, when execute all instruction of chunk in one iter
	// it must close upvalues for that iter
	fc.CloseBlock(-1)

	fc.AddInst(opCreateASbx(OP_JMP, 0, condLabel))
	fc.MarkLabel(endLabel, fc.Inst.LastIndex())
	fc.LeaveBlock(false)
}

func compileForNumberStmt(fc *FunctionContext, stmt *ast.ForNumberStmt) {
	endLabel := fc.NewLabel()
	doLabel := fc.NewLabel()
	fc.EnterBlock(endLabel)
	// counter, end, step is 3 first local vars of the new block
	counter := fc.AddLocalVar(stmt.CounterName)
	end := fc.AddLocalVar("_e_")
	step := fc.AddLocalVar("_sp_")

	slot := fc.StackTop()
	compileExpr(fc, stmt.Start, slot, eOption(1))
	fc.AddInst(opCreateABC(OP_MOVE, counter, slot, 0))

	compileExpr(fc, stmt.End, slot, eOption(1))
	fc.AddInst(opCreateABC(OP_MOVE, end, slot, 0))

	if stmt.Step != nil {
		compileExpr(fc, stmt.Step, slot, eOption(1))
		fc.AddInst(opCreateABC(OP_MOVE, step, slot, 0))
	} else {
		fc.AddInst(opCreateABC(OP_LOADK, step, fc.Consts.IndexOf(KNumber(1)), 0))
	}

	fc.AddInst(opCreateABC(OP_LT, 0, counter, end))
	fc.AddInst(opCreateASbx(OP_JMP, 0, endLabel))

	fc.MarkLabel(doLabel, fc.Inst.LastIndex())
	compileChunk(fc, stmt.Chunk)

	fc.CloseBlock(3) // not close counter, end, step

	// OP_FORLOOP
	/*   A sBx   R(A)+=R(A+2);
	     if R(A) <?= R(A+1) then { pc+=sBx; R(A+3)=R(A) }*/
	fc.AddInst(opCreateASbx(OP_FORLOOP, counter, fc.GetLabelPosition(doLabel)-(fc.Inst.LastIndex()+1)))

	/* fc.AddInst(opCreateABC(OP_ADD, counter, counter, step))
	fc.AddInst(opCreateASbx(OP_JMP, 0, condLabel)) */

	fc.MarkLabel(endLabel, fc.Inst.LastIndex())

	fc.LeaveBlock(true)
}

func compileBreakStmt(fc *FunctionContext, _ *ast.BreakStmt) {
	block := fc.CurBlock
	for block != nil {
		fmt.Println(block)
		if block.NeedClose {
			fc.AddInst(opCreateABC(OP_CLOSE, block.varlist.offset, 0, 0))
		}
		if block.EndLabel != NoBreakLabel {
			fc.AddInst(opCreateASbx(OP_JMP, 0, block.EndLabel))
			return
		}
		block = block.Parent
	}
	panic("invalid break")
}

func compileReturnStmt(fc *FunctionContext, stmt *ast.ReturnStmt) {
	slot := fc.StackTop()
	nexp := len(stmt.Exprs)
	if nexp == 0 {
		fc.AddInst(opCreateABC(OP_RETURN, slot, 1, 0))
		return
	}

	for i := range nexp - 1 {
		slot += compileExpr(fc, stmt.Exprs[i], slot, eOption(1))
	}

	delta := compileExpr(fc, stmt.Exprs[nexp-1], slot, eOption(-1))
	b := nexp + delta
	if delta == 0 {
		b = 0
	}

	fc.AddInst(opCreateABC(OP_RETURN, fc.StackTop(), b, 0))
}
func compileVarDefStmt(fc *FunctionContext, stmt *ast.VarDefStmt) {
	var nvars, nexps = len(stmt.Vars), len(stmt.Exprs)
	slot := fc.StackTop()
	for _, v := range stmt.Vars {
		fc.AddLocalVar(v)
	}

	if nexps == 0 {
		//fmt.Println("set nil..")
		fc.AddInst(opCreateABC(OP_LOADNIL, slot, slot+nvars-1, 0))
		return
	}

	if nexps > nvars {
		panic("too many exprs on rhs")
	}

	right := slot + nvars

	if nvars > nexps {
		fc.AddInst(opCreateABC(OP_LOADNIL, slot+nexps, slot+nvars-1, 0))
	}

	for _, e := range stmt.Exprs {
		compileExpr(fc, e, right, eOption(1))
		right++
	}

	for i := nexps - 1; i >= 0; i-- {
		right--
		fc.AddInst(opCreateABC(OP_MOVE, slot+i, right, 0))
	}
}

func compileFuncDefStmt(fc *FunctionContext, stmt *ast.FuncDefStmt) {
	fc.AddLocalVar(stmt.FuncName)
	funcExpr := &ast.FunctionExpr{
		Params:  stmt.ParList,
		HasVArg: stmt.HasVArg,
		Block:   stmt.Block,
	}
	assignStmt := &ast.AssignStmt{
		Lhs: []ast.Expr{&ast.IdentExpr{Value: stmt.FuncName}},
		Rhs: []ast.Expr{funcExpr},
	}
	compileAssignStmt(fc, assignStmt)
}

func compileListAppendStmt(fc *FunctionContext, stmt *ast.ListAppendStmt) {
	var slot, a, b int
	slot = fc.StackTop()

	compileExprReduceMV(fc, stmt.Object, &slot, &a)
	compileExprReduceMV(fc, stmt.Element, &slot, &b)
	fc.AddInst(opCreateABC(OP_APPEND, a, b, 0))
}

func compileForRangeStmt(fc *FunctionContext, stmt *ast.ForRangeStmt) {
	endLabel := fc.NewLabel()
	doLabel := fc.NewLabel()

	fc.EnterBlock(endLabel)

	slot := fc.StackTop()
	var oslot int
	var used int = 3 // index, value

	compileExprReduceMV(fc, stmt.Object, &slot, &oslot)
	if oslot == slot {
		fc.AddLocalVar("__o")
		used = 4
	}

	lslot := fc.AddLocalVar("__l")

	fc.AddInst(opCreateABC(OP_LEN, lslot, oslot, 0))
	kst0 := fc.Consts.IndexOf(KNumber(0))

	index := fc.AddLocalVar("__i")
	key := fc.AddLocalVar(stmt.Index)
	fc.AddLocalVar(stmt.Value)

	fc.AddInst(opCreateABx(OP_LOADK, index, kst0))
	fc.AddInst(opCreateABC(OP_LT, 0, index, lslot))
	fc.AddInst(opCreateASbx(OP_JMP, 0, endLabel))

	fc.MarkLabel(doLabel, fc.Inst.LastIndex())
	fc.AddInst(opCreateABC(OP_GETFIELD, key, oslot, index))

	//fc.AddInst(opCreateABC(OP_GETTABLE, value, oslot, index))

	compileChunk(fc, stmt.Block)

	fc.CloseBlock(used)

	fc.AddInst(opCreateABC(OP_ADD, index, index, opRkAsk(fc.Consts.IndexOf(KNumber(1)))))
	fc.AddInst(opCreateABC(OP_LT, 1, index, lslot))
	fc.AddInst(opCreateASbx(OP_JMP, 0, doLabel))

	fc.MarkLabel(endLabel, fc.Inst.LastIndex())
	fc.LeaveBlock(true)
}

func compileChunk(fc *FunctionContext, chunk []ast.Stmt) {
	for _, stmt := range chunk {
		compileStmt(fc, stmt)
	}
}

func Compile(chunk []ast.Stmt) *FuncProto {
	funcExpr := &ast.FunctionExpr{
		Params:  []string{},
		HasVArg: true,
		Block:   chunk,
	}
	context := NewFunctionContext(nil, 0, true)
	compileFuncExpr(context, funcExpr)
	return context.Proto
}
