package vm

import (
	"bytes"
	"fmt"

	"github.com/khoakmp/kala/cpi"
)

type UpValue struct {
	next    *UpValue
	index   int
	isClose bool
	value   cpi.KValue
}

func newUpValue(next *UpValue, index int) *UpValue {
	return &UpValue{
		next:    next,
		index:   index,
		isClose: false,
	}
}

func (u *UpValue) Get(stack *StackValue) cpi.KValue {
	if u.isClose {
		return u.value
	}
	return stack.Get(u.index)
}

func (u *UpValue) Set(v cpi.KValue, stack *StackValue) {
	if u.isClose {
		u.value = v
		return
	}
	stack.Set(u.index, v)
}

func (u *UpValue) Close(stack *StackValue) {
	u.isClose = true
	u.value = stack.Get(u.index)
	u.next = nil
}

// Stack of all local variable across func call stack
type StackValue struct {
	array []cpi.KValue
	top   int // the position ready to write new value
}

func newStackValue() *StackValue {
	return &StackValue{
		array: make([]cpi.KValue, 4),
		top:   0,
	}
}

func (s *StackValue) CheckSize(minSize int) {
	if len(s.array) >= minSize {
		return
	}

	arr := make([]cpi.KValue, len(s.array)<<1)
	copy(arr, s.array[:s.top])
	s.array = arr
}

func (s *StackValue) Push(v cpi.KValue) {
	s.CheckSize(s.top + 1)
	s.array[s.top] = v
	s.top++
}
func (s *StackValue) Pop() cpi.KValue {
	if s.top == 0 {
		return cpi.KNil{}
	}
	idx := s.top - 1
	v := s.array[idx]
	s.array[idx] = cpi.KNil{}
	s.top--
	return v
}

func (s *StackValue) Clear(fromIdx, toIdx int) {
	for i := fromIdx; i < toIdx; i++ {
		s.array[i] = nil
	}
}

func (s *StackValue) Set(idx int, v cpi.KValue) {
	s.CheckSize(idx + 1)
	s.array[idx] = v

	if s.top <= idx {
		s.top = idx + 1
	}
}

func (s *StackValue) Get(idx int) cpi.KValue {
	if s.top <= idx {
		return cpi.KNil{}
	}
	return s.array[idx]
}

func (s *StackValue) CopyRange(fromIdx, num int) []cpi.KValue {
	ans := make([]cpi.KValue, num)
	s.CheckSize(fromIdx + num)
	copy(ans, s.array[fromIdx:fromIdx+num])
	return ans
}

func (s *StackValue) MoveRange(fromIdx, toIdx, num int) {
	msize := max(fromIdx, toIdx) + num
	s.CheckSize(msize)
	arr := s.CopyRange(fromIdx, num)
	s.SetRange(toIdx, arr)
}

func (s *StackValue) SetRange(index int, arr []cpi.KValue) {
	s.CheckSize(index + len(arr))
	copy(s.array[index:], arr)
	s.top = max(s.top, index+len(arr))
}

type ClosureFunc struct {
	IsGlobal bool
	Proto    *cpi.FuncProto
	Upvalues []*UpValue
	GF       GlobalFunc
}

func (c *ClosureFunc) Type() int {
	return cpi.KTypeFunction
}

func (c *ClosureFunc) Str() string {
	return "closure"
}

func NewLocalClosure(proto *cpi.FuncProto) *ClosureFunc {
	return &ClosureFunc{
		IsGlobal: false,
		Proto:    proto,
		Upvalues: make([]*UpValue, proto.NumUpvalues),
	}
}

func NewGlobalClosure(gf GlobalFunc) *ClosureFunc {
	return &ClosureFunc{
		IsGlobal: true,
		Proto:    nil,
		Upvalues: nil,
		GF:       gf,
	}
}

func EmbeddedPrint(s *RuntimeState) {
	cf := s.currentFrame
	buffer := bytes.NewBuffer(nil)
	for i := range cf.NumArg {
		r := cf.LocalBase + i
		v := s.stackValue.Get(r)
		buffer.WriteString(v.Str())
		buffer.WriteByte(' ')
	}
	fmt.Println(buffer.String())
	cf.NumRetValue = 0
}

type GlobalFunc func(s *RuntimeState)

type CallFrame struct {
	Base        int
	LocalBase   int
	ReturnBase  int
	Closure     *ClosureFunc
	PC          int
	NumArg      int
	NumRetValue int
}

func newCallFrame(base, localBase, retBase, numRetValue int, closure *ClosureFunc) *CallFrame {
	return &CallFrame{
		Base:        base,
		LocalBase:   localBase,
		ReturnBase:  retBase,
		Closure:     closure,
		PC:          0,
		NumRetValue: numRetValue,
	}
}

type StackCallFrame struct {
	array []*CallFrame
}

func (s *StackCallFrame) Last() *CallFrame {
	idx := len(s.array) - 1
	if idx < 0 {
		return nil
	}
	return s.array[idx]
}

func (s *StackCallFrame) Push(cf *CallFrame) (index int) {
	index = len(s.array)
	s.array = append(s.array, cf)
	return
}

// this cause the stackCallFrame does not keep the pointer to the current frame
func (s *StackCallFrame) Pop() {
	l := len(s.array)
	if l == 0 {
		return
	}
	s.array[l-1] = nil
	s.array = s.array[:l-1]
}

type RuntimeState struct {
	stackValue     *StackValue
	stackCallFrame StackCallFrame
	currentFrame   *CallFrame
	firstUV        *UpValue
	Global         cpi.KDict
}

func (s *RuntimeState) CallGFunction() {
	cf := s.currentFrame
	cf.Closure.GF(s)
	s.stackCallFrame.Pop()
	s.currentFrame = s.stackCallFrame.Last()
}

func (s *RuntimeState) LastFrame() *CallFrame {
	return s.stackCallFrame.Last()
}

func CreateGlobal() cpi.KDict {
	dict := cpi.NewKDict(1)
	dict.SetField("print", NewGlobalClosure(EmbeddedPrint))
	return dict
}

func NewRState() *RuntimeState {

	return &RuntimeState{
		stackValue: newStackValue(),
		stackCallFrame: StackCallFrame{
			array: make([]*CallFrame, 0, 1),
		},
		currentFrame: nil,
		firstUV:      nil,
		Global:       CreateGlobal(),
	}
}
func (s *RuntimeState) CloseUpvalues(startIndex int) {
	//fmt.Println("Close upvalues from", startIndex)
	prev := s.firstUV
	closeFn := func(p *UpValue) {
		for p != nil {
			next := p.next
			p.Close(s.stackValue)
			p = next
		}
	}
	if prev == nil || prev.index >= startIndex {
		s.firstUV = nil
		closeFn(prev)
		return
	}
	for {
		cur := prev.next
		if cur == nil || cur.index >= startIndex {
			prev.next = nil
			closeFn(cur)
			return
		}
		prev = cur
	}
}

func (s *RuntimeState) FindUpValue(index int) *UpValue {
	if s.firstUV == nil {
		s.firstUV = newUpValue(nil, index)
		return s.firstUV
	}

	if s.firstUV.index == index {
		return s.firstUV
	}

	if s.firstUV.index > index {
		node := newUpValue(s.firstUV, index)
		s.firstUV = node
		return node
	}
	prev := s.firstUV

	for cur := prev.next; cur != nil; cur = prev.next {
		if cur.index == index {
			return cur
		}
		if cur.index > index {
			node := newUpValue(cur, index)
			prev.next = node
			return node
		}
		prev = cur
	}
	node := newUpValue(nil, index)
	prev.next = node
	return node
}
func opGetArgA(inst uint32) int {
	return int(inst>>18) & 0xff
}

func opGetArgB(inst uint32) int {
	return int(inst & 0x1ff)
}

func opGetArgC(inst uint32) int {
	return int(inst>>9) & 0x1ff
}

func opIsK(value int) bool {
	return bool((value & 256) != 0)
}

func opIndexK(value int) int {
	return value & ^256
}

func opGetArgBx(inst uint32) int {
	return int(inst & 0x3ffff)
}
func opGetOpCode(inst uint32) int {
	return int(inst >> 26)
}

func opGetArgSbx(inst uint32) int {
	return opGetArgBx(inst) - 131071
}
