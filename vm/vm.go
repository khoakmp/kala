package vm

import (
	"github.com/khoakmp/kala/cpi"
)

func Prepare(proto *cpi.FuncProto) *RuntimeState {
	state := NewRState()
	closure := NewLocalClosure(proto)

	callframe := newCallFrame(0, 0, 0, -1, closure)
	state.stackCallFrame.Push(callframe)
	state.currentFrame = callframe
	return state
}

func Run(proto *cpi.FuncProto) {
	state := Prepare(proto)
	frame := state.currentFrame

	for ; frame != nil; frame = state.currentFrame {
		inst := frame.Closure.Proto.InstList.At(frame.PC)
		frame.PC++
		execFunc[opGetOpCode(inst)](state, inst)
	}
}

// this function is just used for testing
func (s *RuntimeState) Run(uptoPC int) {
	rootFrame := s.currentFrame
	for frame := s.currentFrame; frame != nil; frame = s.currentFrame {
		if frame == rootFrame && frame.PC == uptoPC {
			return
		}
		inst := frame.Closure.Proto.InstList.At(frame.PC)
		frame.PC++
		execFunc[opGetOpCode(inst)](s, inst)
	}
}

var execFunc [44]func(s *RuntimeState, inst uint32)

func init() {
	execFunc[0] = EXEC_OP_MOVE
	execFunc[1] = nil // OP_MOVEN
	execFunc[2] = EXEC_OP_LOADK
	execFunc[3] = EXEC_OP_LOADBOOL
	execFunc[4] = EXEC_OP_LOADNIL
	execFunc[5] = EXEC_OP_GETUPVAL
	execFunc[6] = EXEC_OP_GETGLOBAL
	execFunc[7] = EXEC_OP_GETTABLE
	execFunc[8] = EXEC_OP_GETTABLEKS
	execFunc[9] = EXEC_OP_SETGLOBAL
	execFunc[10] = EXEC_OP_SETUPVAL
	execFunc[11] = EXEC_OP_SETTABLE
	execFunc[12] = EXEC_OP_SETTABLEKS
	execFunc[13] = EXEC_OP_NEWTABLE
	execFunc[14] = nil // OP_SELF

	for i := 15; i < 20; i++ {
		execFunc[i] = EXEC_OP_Arithmetic
	}
	execFunc[20] = nil
	execFunc[21] = EXEC_OP_UNM
	execFunc[22] = EXEC_OP_NOT
	execFunc[23] = EXEC_OP_LEN
	execFunc[24] = EXEC_OP_CONCAT
	execFunc[25] = EXEC_OP_JMP
	for i := 26; i <= 28; i++ {
		execFunc[i] = EXEC_OP_Relational
	}
	execFunc[29] = EXEC_OP_TEST
	execFunc[30] = nil
	execFunc[31] = EXEC_OP_CALL
	execFunc[32] = nil
	execFunc[33] = EXEC_OP_RETURN
	execFunc[34] = EXEC_OP_FORLOOP
	execFunc[35] = nil // OP_FORPREP
	execFunc[36] = nil // OP_TFORLOOP
	execFunc[37] = EXEC_OP_SETLIST
	execFunc[38] = EXEC_OP_CLOSE
	execFunc[39] = EXEC_OP_CLOSURE
	execFunc[40] = EXEC_OP_VARARG
	execFunc[41] = EXEC_OP_NOP
	execFunc[42] = EXEC_OP_APPEND
	execFunc[43] = EXEC_OP_GETFIELD
}

func EXEC_OP_MOVE(s *RuntimeState, inst uint32) {
	a, b := opGetArgA(inst), opGetArgB(inst)
	base := s.currentFrame.LocalBase
	ra, rb := base+a, base+b
	stack := s.stackValue
	v := stack.Get(rb)
	stack.Set(ra, v)
}

func EXEC_OP_LOADK(s *RuntimeState, inst uint32) {
	a, b := opGetArgA(inst), opGetArgB(inst)
	cf := s.currentFrame
	lbase := cf.LocalBase
	stack := s.stackValue
	ra := lbase + a
	v := cf.Closure.Proto.Consts.GetAt(b)
	stack.Set(ra, v)
}

func EXEC_OP_GETGLOBAL(s *RuntimeState, inst uint32) {
	// A Bx    R(A) := Gbl[Kst(Bx)]
	a, bx := opGetArgA(inst), opGetArgBx(inst)
	cf := s.currentFrame
	ra := cf.LocalBase + a
	field := cf.Closure.Proto.StringConsts[bx]
	v := s.Global.GetField(field)
	s.stackValue.Set(ra, v)
}

func EXEC_OP_SETGLOBAL(s *RuntimeState, inst uint32) {
	// A Bx    Gbl[Kst(Bx)] := R(A)
	a, bx := opGetArgA(inst), opGetArgBx(inst)
	cf := s.currentFrame
	ra := cf.LocalBase + a
	field := cf.Closure.Proto.StringConsts[bx]
	s.Global.SetField(field, s.stackValue.Get(ra))
}

func EXEC_OP_GETTABLE(s *RuntimeState, inst uint32) {
	// R[a] = R[b][Rk[c]]
	a, b, c := opGetArgA(inst), opGetArgB(inst), opGetArgC(inst)

	cf := s.currentFrame
	ra, rb := cf.LocalBase+a, cf.LocalBase+b

	stack := s.stackValue
	v := stack.Get(rb)
	var key, value cpi.KValue
	key = s.GetValue(c)

	switch v := v.(type) {
	case cpi.KDict:
		switch key := key.(type) {
		case cpi.KString:
			value = v.GetField(string(key))
		case cpi.KNumber:
			value = v.GetAt(int(key))
		default:
			panic("wrong type")
		}

		//value = v.GetField(string(key.(cpi.KString)))
	case cpi.KList:
		n, ok := key.(cpi.KNumber)
		if !ok {
			panic("wrong type")
		}
		value = v.GetAt(int(n))
	}
	stack.Set(ra, value)
}

func EXEC_OP_GETTABLEKS(s *RuntimeState, inst uint32) {
	//  A B C   R(A) := R(B)[RK(C)] ; RK(C) is constant string
	a, b, c := opGetArgA(inst), opGetArgB(inst), opGetArgC(inst)
	cf := s.currentFrame
	ra, rb := cf.LocalBase+a, cf.LocalBase+b

	stack := s.stackValue
	dict := stack.Get(rb).(cpi.KDict)
	key := cf.Closure.Proto.Consts.GetAt(c).(cpi.KString)

	stack.Set(ra, dict.GetField(string(key)))
}

func EXEC_OP_SETTABLE(s *RuntimeState, inst uint32) {
	// R[a][RK[b]] = RK[c]
	a, b, c := opGetArgA(inst), opGetArgB(inst), opGetArgC(inst)
	cf := s.currentFrame
	ra := cf.LocalBase + a
	stack := s.stackValue
	var key, value cpi.KValue
	key = s.GetValue(b)
	value = s.GetValue(c)
	table := stack.Get(ra)

	switch table := table.(type) {
	case cpi.KDict:
		table.SetField(string(key.(cpi.KString)), value)
	case cpi.KList:
		table.SetAt(int(key.(cpi.KNumber)), value)
	}
}

func EXEC_OP_SETTABLEKS(s *RuntimeState, inst uint32) {
	// A B C   R(A)[RK(B)] := RK(C) ; RK(B) is constant string
	a, b, c := opGetArgA(inst), opGetArgB(inst), opGetArgC(inst)
	cf := s.currentFrame
	stack := s.stackValue
	ra := cf.LocalBase + a

	v := s.GetValue(c)
	key := cf.Closure.Proto.Consts.GetAt(b).(cpi.KString)
	dict := stack.Get(ra).(cpi.KDict)
	dict.SetField(string(key), v)
}

func EXEC_OP_GETUPVAL(s *RuntimeState, inst uint32) {
	// R(A) := UpValue[B]
	a, b := opGetArgA(inst), opGetArgB(inst)
	cf := s.currentFrame
	ra := cf.LocalBase + a
	stack := s.stackValue
	v := cf.Closure.Upvalues[b].Get(stack)
	stack.Set(ra, v)
}

func EXEC_OP_SETUPVAL(s *RuntimeState, inst uint32) {
	//UpValue[B] := R(A)
	a, b := opGetArgA(inst), opGetArgB(inst)
	stack := s.stackValue
	cf := s.currentFrame
	ra := cf.LocalBase + a
	v := stack.Get(ra)
	cf.Closure.Upvalues[b].Set(v, stack)
}

func EXEC_OP_CLOSE(s *RuntimeState, inst uint32) {
	a := opGetArgA(inst)
	/* cf := s.currentFrame
	ra := cf.LocalBase + a */
	idx := s.currentFrame.LocalBase + a
	s.CloseUpvalues(idx)
	s.stackValue.Clear(idx, s.stackValue.top)
}

func EXEC_OP_CLOSURE(s *RuntimeState, inst uint32) {
	/*   A Bx    R(A) := closure(KPROTO[Bx] R(A) ... R(A+n))  */
	a, bx := opGetArgA(inst), opGetArgBx(inst)
	cf := s.currentFrame
	ra := cf.LocalBase + a
	childProto := cf.Closure.Proto.FuncProtos[bx]
	closure := NewLocalClosure(childProto)
	instList := cf.Closure.Proto.InstList
	//fmt.Println("num upvalues:", childProto.NumUpvalues)
	for i := range childProto.NumUpvalues {
		inst := instList.At(cf.PC)
		cf.PC++
		b := opGetArgB(inst)
		op := opGetOpCode(inst)

		switch op {
		case cpi.OP_MOVE:
			rb := cf.LocalBase + b
			//fmt.Printf("upvalue[%d] is reg %d\n", i, rb)
			closure.Upvalues[i] = s.FindUpValue(rb)
		case cpi.OP_GETUPVAL:
			closure.Upvalues[i] = cf.Closure.Upvalues[b]
		}
	}
	s.stackValue.Set(ra, closure)
}

func (s *RuntimeState) GetValue(rk int) cpi.KValue {
	cf := s.currentFrame
	if opIsK(rk) {
		return cf.Closure.Proto.Consts.GetAt(opIndexK(rk))
	}
	return s.stackValue.Get(rk + cf.LocalBase)
}

// ADD, SUB, MUL, DIV, MOD
func EXEC_OP_Arithmetic(s *RuntimeState, inst uint32) {
	op := opGetOpCode(inst)
	a, b, c := opGetArgA(inst), opGetArgB(inst), opGetArgC(inst)
	cf := s.currentFrame
	ra := cf.LocalBase + a

	var lval, rval cpi.KValue
	lval = s.GetValue(b)
	rval = s.GetValue(c)

	if lval.Type() != cpi.KTypeNumber || rval.Type() != cpi.KTypeNumber {
		panic("wrong type")
	}
	var result cpi.KNumber
	switch op {
	case cpi.OP_ADD:
		result = lval.(cpi.KNumber) + rval.(cpi.KNumber)
	case cpi.OP_SUB:
		result = lval.(cpi.KNumber) - rval.(cpi.KNumber)
	case cpi.OP_MUL:
		result = lval.(cpi.KNumber) * rval.(cpi.KNumber)
	case cpi.OP_DIV:
		result = lval.(cpi.KNumber) / rval.(cpi.KNumber)
	case cpi.OP_MOD:
		result = cpi.KNumber(int(lval.(cpi.KNumber)) % int(rval.(cpi.KNumber)))
	}
	s.stackValue.Set(ra, result)
}

func EXEC_OP_Relational(s *RuntimeState, inst uint32) {
	/*        A B C   if ((RK(B) == RK(C)) ~= A) then pc++            */
	a, b, c := opGetArgA(inst), opGetArgB(inst), opGetArgC(inst)
	op := opGetOpCode(inst)

	lval := s.GetValue(b)
	rval := s.GetValue(c)
	//fmt.Println("compare:", cpi.TypeNames[lval.Type()], "and", cpi.TypeNames[rval.Type()])

	var r bool = a == 1
	switch op {
	case cpi.OP_EQ:
		if (lval == rval) != r {
			s.currentFrame.PC++
		}
	case cpi.OP_LT:
		if lval.Type() != cpi.KTypeNumber || rval.Type() != cpi.KTypeNumber {
			panic("wrong type")
		}
		if (lval.(cpi.KNumber) < rval.(cpi.KNumber)) != r {
			s.currentFrame.PC++
		}
	case cpi.OP_LE:
		if lval.Type() != cpi.KTypeNumber || rval.Type() != cpi.KTypeNumber {
			panic("wrong type")
		}
		if (lval.(cpi.KNumber) <= rval.(cpi.KNumber)) != r {
			s.currentFrame.PC++
		}
	}
}

func EXEC_OP_JMP(s *RuntimeState, inst uint32) {
	sbx := opGetArgSbx(inst)
	s.currentFrame.PC += sbx
}

func EXEC_OP_FORLOOP(s *RuntimeState, inst uint32) {
	a, sbx := opGetArgA(inst), opGetArgSbx(inst)
	cf := s.currentFrame
	ra := cf.LocalBase + a
	stack := s.stackValue

	step := int(stack.Get(ra + 2).(cpi.KNumber))
	counter := int(stack.Get(ra).(cpi.KNumber)) + step
	stack.Set(ra, cpi.KNumber(counter))
	limit := int(stack.Get(ra + 1).(cpi.KNumber))
	if counter < limit {
		cf.PC += sbx
	}
}

func EXEC_OP_LOADBOOL(s *RuntimeState, inst uint32) {
	a, b, c := opGetArgA(inst), opGetArgB(inst), opGetArgC(inst)
	var val bool = b == 1
	cf := s.currentFrame
	s.stackValue.Set(cf.LocalBase+a, cpi.KBool(val))
	if c == 1 {
		cf.PC++
	}
}

func EXEC_OP_LOADNIL(s *RuntimeState, inst uint32) {
	a, b := opGetArgA(inst), opGetArgB(inst)
	cf := s.currentFrame
	ra, rb := cf.LocalBase+a, cf.LocalBase+b
	for i := ra; i <= rb; i++ {
		s.stackValue.Set(i, cpi.KNil{})
	}
}

func EXEC_OP_NEWTABLE(s *RuntimeState, inst uint32) {
	a, b, c := opGetArgA(inst), opGetArgB(inst), opGetArgC(inst)
	cf := s.currentFrame
	ra := cf.LocalBase + a

	if c > 0 {
		dict := cpi.NewKDict(c)
		s.stackValue.Set(ra, dict)
		return
	}
	list := cpi.NewKList(b)
	s.stackValue.Set(ra, list)
}

func EXEC_OP_LEN(s *RuntimeState, inst uint32) {
	// R(A) := length of R(B)
	a, b := opGetArgA(inst), opGetArgB(inst)
	cf := s.currentFrame
	ra, rb := cf.LocalBase+a, cf.LocalBase+b
	v := s.stackValue.Get(rb)

	var l int
	switch v := v.(type) {
	case cpi.KDict:
		l = v.Len()
	case cpi.KList:
		l = v.Len()
	default:
		panic("wrong type")
	}
	s.stackValue.Set(ra, cpi.KNumber(l))
}

func EXEC_OP_NOT(s *RuntimeState, inst uint32) {
	// R[a] = ! R[b]
	a, b := opGetArgA(inst), opGetArgB(inst)
	cf := s.currentFrame
	ra, rb := cf.LocalBase+a, cf.LocalBase+b
	v := s.stackValue.Get(rb)
	if v, ok := v.(cpi.KBool); ok {
		s.stackValue.Set(ra, !v)
		return
	}
	panic("wrong type")
}

func EXEC_OP_UNM(s *RuntimeState, inst uint32) {
	// R[a] = -R[b]
	a, b := opGetArgA(inst), opGetArgB(inst)
	cf := s.currentFrame
	ra, rb := cf.LocalBase+a, cf.LocalBase+b
	v, ok := s.stackValue.Get(rb).(cpi.KNumber)
	if !ok {
		panic("wrong type")
	}
	s.stackValue.Set(ra, -v)
}

func EXEC_OP_TEST(s *RuntimeState, inst uint32) {
	// A C     if not (R(A) <=> C) then pc++
	// R[a] != c => pc++
	a, c := opGetArgA(inst), opGetArgC(inst)
	cf := s.currentFrame
	va := s.stackValue.Get(cf.LocalBase + a)
	var bv bool
	switch v := va.(type) {
	case cpi.KBool:
		bv = bool(v)
	case cpi.KNumber:
		bv = int(v) != 0
	}
	if bv != (c == 1) {
		cf.PC++
	}
}

func EXEC_OP_CONCAT(s *RuntimeState, inst uint32) {
	//  A B C   R(A) := R(B).. ... ..R(C)
	a, b, c := opGetArgA(inst), opGetArgB(inst), opGetArgC(inst)
	cf := s.currentFrame
	ra, rb, rc := cf.LocalBase+a, cf.LocalBase+b, cf.LocalBase+c
	vb := s.stackValue.Get(rb).(cpi.KString)
	vc := s.stackValue.Get(rc).(cpi.KString)
	s.stackValue.Set(ra, vb+vc)
}

func EXEC_OP_CALL(s *RuntimeState, inst uint32) {
	a, b, c := opGetArgA(inst), opGetArgB(inst), opGetArgC(inst)
	cf := s.currentFrame
	stack := s.stackValue
	ra := cf.LocalBase + a
	closure, ok := stack.Get(ra).(*ClosureFunc)
	if !ok {
		panic("wrong type: call non-function type")
	}

	var narg int = b - 1
	if narg < 0 {
		narg = stack.top - (ra + 1)
	}

	callFrame := newCallFrame(ra, ra+1, ra, c-1, closure)
	callFrame.NumArg = narg

	s.stackCallFrame.Push(callFrame)

	if closure.IsGlobal {
		s.currentFrame = callFrame
		s.CallGFunction()
		return
	}

	proto := closure.Proto
	localbase := ra + 1

	npar := proto.NumParams
	if narg < npar {
		panic("wrong number arguments")
	}

	var nvarg int = 0

	if proto.HasVarg {
		nvarg = narg - npar
	}

	if nvarg > 0 {
		vargs := stack.CopyRange(localbase+npar, nvarg)
		stack.MoveRange(localbase, localbase+nvarg, npar)
		stack.SetRange(localbase, vargs)
		callFrame.LocalBase = localbase + nvarg
		list := cpi.NewKList(nvarg)
		list.AppendArray(vargs)
		stack.Set(callFrame.LocalBase+npar, list)

	} else if proto.HasVarg {
		list := cpi.NewKList(0)
		stack.Set(localbase+npar, list)
	}

	s.currentFrame = callFrame
}

func EXEC_OP_RETURN(s *RuntimeState, inst uint32) {
	a, b := opGetArgA(inst), opGetArgB(inst)
	cf := s.currentFrame
	ra := cf.LocalBase + a

	stack := s.stackValue
	var nret int = b - 1

	if nret < 0 {
		nret = stack.top - ra
	}

	if nret < cf.NumRetValue {
		panic("wrong number return value")
	}

	if cf.NumRetValue >= 0 {
		nret = cf.NumRetValue
	}
	//fmt.Println("nret:", nret, "ra:", ra)
	s.CloseUpvalues(cf.LocalBase)

	stack.MoveRange(ra, cf.ReturnBase, nret)

	stack.Clear(cf.ReturnBase+nret, stack.top)
	stack.top = cf.ReturnBase + nret

	s.stackCallFrame.Pop()
	s.currentFrame = s.stackCallFrame.Last()
}

func EXEC_OP_SETLIST(s *RuntimeState, inst uint32) {
	a, b := opGetArgA(inst), opGetArgB(inst)
	cf := s.currentFrame
	ra := cf.LocalBase + a
	stack := s.stackValue
	list, ok := stack.Get(ra).(cpi.KList)
	if !ok {
		list = cpi.NewKList(b)
	}
	for i := 0; i < b; i++ {
		list.Append(stack.Get(ra + 1 + i))
	}
	stack.Set(ra, list)
}

func EXEC_OP_APPEND(s *RuntimeState, inst uint32) {
	a, b := opGetArgA(inst), opGetArgB(inst)
	cf := s.currentFrame
	ra, rb := cf.LocalBase+a, cf.LocalBase+b
	list := s.stackValue.Get(ra).(cpi.KList)
	v := s.stackValue.Get(rb)
	list.Append(v)
}
func EXEC_OP_GETFIELD(s *RuntimeState, inst uint32) {
	a, b, c := opGetArgA(inst), opGetArgB(inst), opGetArgC(inst)
	cf := s.currentFrame
	ra, rb, rc := cf.LocalBase+a, cf.LocalBase+b, cf.LocalBase+c
	index := int(s.stackValue.Get(rc).(cpi.KNumber))
	v := s.stackValue.Get(rb)
	switch v := v.(type) {
	case cpi.KDict:
		k, value := v.GetKeyValue(index)
		s.stackValue.Set(ra, cpi.KString(k))
		s.stackValue.Set(ra+1, value)
	case cpi.KList:
		value := v.GetAt(index)
		s.stackValue.Set(ra, cpi.KNumber(index))
		s.stackValue.Set(ra+1, value)
	default:
		panic("wrong type")
	}
}
func EXEC_OP_VARARG(s *RuntimeState, inst uint32) {
	// A B     R(A) R(A+1) ... R(A+B-1) = vararg

}

func EXEC_OP_NOP(s *RuntimeState, inst uint32) {}
