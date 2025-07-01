package cpi

import "fmt"

const (
	LocalVarListInitSize = 4
)
const NoBreakLabel = 0

type VarList struct {
	names  []string
	offset int
}

func (l *VarList) Len() int {
	return len(l.names)
}

func newVarlist(offset int, cap int) *VarList {
	return &VarList{
		names:  make([]string, 0, cap),
		offset: offset,
	}
}

func (l *VarList) GetUnique(name string) int {
	for i, v := range l.names {
		if v == name {
			return i
		}
	}
	return l.Add(name)
}

func (l *VarList) Add(name string) int {
	l.names = append(l.names, name)
	return len(l.names) - 1 + l.offset
}

func (l *VarList) List() []string {
	return l.names
}

type Block struct {
	Parent    *Block
	EndLabel  int
	varlist   VarList
	NeedClose bool
}

func newBlock(endLabel int, offset int, parent *Block) *Block {
	return &Block{
		varlist: VarList{
			names:  make([]string, 0, LocalVarListInitSize),
			offset: offset,
		},
		Parent:   parent,
		EndLabel: endLabel,
	}
}

func (b *Block) AddLocalVar(name string) int {
	b.varlist.names = append(b.varlist.names, name)
	return b.varlist.offset + len(b.varlist.names) - 1
}

type InstructionList struct {
	insts []uint32
}

func newInstructionList(cap int) *InstructionList {
	return &InstructionList{
		insts: make([]uint32, 0, cap),
	}
}

func (l *InstructionList) LastIndex() int {
	return len(l.insts) - 1
}

func (l *InstructionList) Last() uint32 {
	if len(l.insts) == 0 {
		return 0
	}
	return l.insts[len(l.insts)-1]
}

func (i *InstructionList) List() []uint32 {
	return i.insts
}

func (i *InstructionList) Add(ins uint32) {
	i.insts = append(i.insts, ins)
}

func (i *InstructionList) At(idx int) uint32 {
	return i.insts[idx]
}

func (l *InstructionList) AddNil(a, b int) {
	last := l.Last()
	if opGetOpCode(last) == OP_LOADNIL {
		if opGetArgB(last) == a-1 {
			opSetArgB(&last, b)
			return
		}
	}
	l.Add(opCreateABC(OP_LOADNIL, a, b, 0))
}

func (l *InstructionList) SetOpCode(position int, opcode int) {
	opSetOpCode(&l.insts[position], opcode)
}

func (l *InstructionList) SetSbx(position int, sbx int) {
	opSetArgSbx(&l.insts[position], sbx)
}

type Constansts struct {
	data []KValue
}

func newConstants(cap int) *Constansts {
	return &Constansts{
		data: make([]KValue, 0, cap),
	}
}

func (c *Constansts) GetAt(index int) KValue {
	return c.data[index]
}

func (c *Constansts) IndexOf(v KValue) int {
	for i, val := range c.data {
		if compareKValue(v, val) {
			return i
		}
	}
	c.data = append(c.data, v)
	return len(c.data) - 1
}

func (c *Constansts) Len() int {
	return len(c.data)
}

func (c *Constansts) Print() {
	for idx, v := range c.data {
		fmt.Printf("Kst[%d], Type: %s, Value: %s\n", idx, TypeNames[v.Type()], v.Str())
	}
}

type FuncProto struct {
	Consts           *Constansts
	InstList         *InstructionList
	NumParams        int
	NumUpvalues      int
	FuncProtos       []*FuncProto
	HasVarg          bool
	NumUsedRegisters uint8
	StringConsts     []string
}

func (p *FuncProto) AddChildProto(proto *FuncProto) {
	p.FuncProtos = append(p.FuncProtos, proto)
}

func newFuncProto(numParam int, hasVarg bool) *FuncProto {
	return &FuncProto{
		Consts:      nil,
		InstList:    nil,
		NumParams:   numParam,
		NumUpvalues: 0,
		HasVarg:     hasVarg,
		FuncProtos:  make([]*FuncProto, 0),
	}
}

type FunctionContext struct {
	Parent         *FunctionContext
	labelCnt       int
	Proto          *FuncProto
	CurBlock       *Block
	stackTop       int
	Inst           *InstructionList
	Consts         *Constansts
	Upvalues       *VarList    // upvalues of this function refer to outer functions context
	LabelPositions map[int]int // map from label to instruction position
}

func (fc *FunctionContext) GetLabelPosition(label int) int {
	return fc.LabelPositions[label]
}

func (fc *FunctionContext) MarkLabel(label, instPosition int) {
	fc.LabelPositions[label] = instPosition
}

func (fc *FunctionContext) NewLabel() int {
	fc.labelCnt++
	return fc.labelCnt
}

func (fc *FunctionContext) SetStackTop(v int) {
	fc.stackTop = v
}

func (fc *FunctionContext) AddInst(i uint32) {
	fc.Inst.Add(i)
}

func (fc *FunctionContext) AddLocalVar(name string) int {
	l := len(fc.CurBlock.varlist.names)
	fc.CurBlock.varlist.Add(name)
	//fmt.Println("var", name, "slot:", fc.CurBlock.varlist.offset+l)
	fc.SetStackTop(fc.stackTop + 1)
	return fc.CurBlock.varlist.offset + l
}

func (fc *FunctionContext) FindLocalVar(name string) int {
	block := fc.CurBlock
	for block != nil {
		for idx, v := range block.varlist.names {
			if v == name {
				return idx + block.varlist.offset
			}
		}
		block = block.Parent
	}
	return -1
}

func (fc *FunctionContext) FindLocaVarAndBlock(name string) (slot int, block *Block) {
	block = fc.CurBlock
	for block != nil {
		for idx, v := range block.varlist.names {
			if v == name {
				slot = idx + block.varlist.offset
				return
			}
		}
		block = block.Parent
	}
	slot = -1
	return
}

func NewFunctionContext(parent *FunctionContext, numParam int, hasVarg bool) *FunctionContext {
	fc := &FunctionContext{
		stackTop:       0,
		labelCnt:       0,
		Parent:         parent,
		Proto:          newFuncProto(numParam, hasVarg),
		CurBlock:       newBlock(NoBreakLabel, 0, nil),
		Inst:           newInstructionList(10),
		Consts:         newConstants(0),
		Upvalues:       newVarlist(0, 0),
		LabelPositions: make(map[int]int),
	}

	return fc
}

func (fc *FunctionContext) StackTop() int {
	return fc.stackTop
}

func (fc *FunctionContext) EnterBlock(endLabel int) {
	fc.CurBlock = newBlock(endLabel, fc.stackTop, fc.CurBlock)
}

// close all local variable of block that refered by other closure
func (fc *FunctionContext) CloseBlock(from int) {
	if !fc.CurBlock.NeedClose {
		return
	}
	a := fc.CurBlock.varlist.offset
	if from != -1 {
		a += from
	}
	fc.Inst.Add(opCreateABC(OP_CLOSE, a, 0, 0))
}

func (fc *FunctionContext) LeaveBlock(closeUpvalue bool) {
	if closeUpvalue {
		fc.CloseBlock(-1)
	}
	fc.SetStackTop(fc.CurBlock.varlist.offset)
	fc.CurBlock = fc.CurBlock.Parent
}

func patchCode(context *FunctionContext) {
	maxreg := 1
	if np := int(context.Proto.NumParams); np > 1 {
		maxreg = np
	}
	code := context.Inst.List()
	for pc := 0; pc < len(code); pc++ {
		inst := code[pc]
		curop := opGetOpCode(inst)
		switch curop {
		case OP_CLOSURE:
			pc += int(context.Proto.FuncProtos[opGetArgBx(inst)].NumUpvalues)
			continue
		case OP_SETGLOBAL, OP_SETUPVAL, OP_EQ, OP_LT, OP_LE, OP_TEST,
			OP_TAILCALL, OP_RETURN, OP_FORPREP, OP_FORLOOP, OP_TFORLOOP,
			OP_SETLIST, OP_CLOSE:
			/* nothing to do */
		case OP_CALL:
			if reg := opGetArgA(inst) + opGetArgC(inst) - 2; reg > maxreg {
				maxreg = reg
			}
		case OP_VARARG:
			if reg := opGetArgA(inst) + opGetArgB(inst) - 1; reg > maxreg {
				maxreg = reg
			}
		case OP_SELF:
			if reg := opGetArgA(inst) + 1; reg > maxreg {
				maxreg = reg
			}
		case OP_LOADNIL:
			if reg := opGetArgB(inst); reg > maxreg {
				maxreg = reg
			}
		case OP_JMP:
			distance := 0
			count := 0
			for jmp := inst; opGetOpCode(jmp) == OP_JMP && count < 5; jmp = context.Inst.At(pc + distance + 1) {
				d := context.GetLabelPosition(opGetArgSbx(jmp)) - pc
				if d > opMaxArgSbx {
					if distance == 0 {
						panic("Jump too long")
					}
					break
				}
				distance = d
				count++
			}
			if distance == 0 {
				context.Inst.SetOpCode(pc, OP_NOP)
			} else {
				context.Inst.SetSbx(pc, distance)
			}
		default:
			if reg := opGetArgA(inst); reg > maxreg {
				maxreg = reg
			}
		}

	}
	maxreg++
	if maxreg > 255 {
		panic("exceed max stack slot")
	}
	context.Proto.NumUsedRegisters = uint8(maxreg)
}
