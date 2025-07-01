package vm

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/khoakmp/kala/cpi"
	"github.com/khoakmp/kala/parse"
	"github.com/stretchr/testify/assert"
)

func compile(src string) *cpi.FuncProto {
	rd := bytes.NewReader([]byte(src))
	chunk, err := parse.Parse(rd, "")
	if err != nil {
		panic(err)
	}
	return cpi.Compile(chunk)
}

func TestCopyValue(t *testing.T) {
	t.Run("move_bool_nil_string", func(t *testing.T) {
		src := `
		var a,b = false, true
		a = b 		
		var c = nil 
		var d = c 
		c = "kmp"
		var e = c
		`
		proto := compile(src)
		state := Prepare(proto)
		state.Run(proto.InstList.LastIndex())
		stack := state.stackValue
		va, vb := stack.Get(1), stack.Get(2)
		vc, vd, ve := stack.Get(3), stack.Get(4), stack.Get(5)
		assert.Equal(t, va, vb)
		assert.Equal(t, true, bool(va.(cpi.KBool)))
		//assert.Equal(t, vc, vd)
		assert.Equal(t, cpi.KTypeNil, vd.Type())
		assert.Equal(t, cpi.KTypeString, vc.Type())
		assert.Equal(t, ve, vc)
	})

	t.Run("move_number", func(t *testing.T) {
		src := `
			var a,b = 12, 23 
			a = b 		
		`
		proto := compile(src)
		state := Prepare(proto)
		state.Run(proto.InstList.LastIndex())
		stack := state.stackValue
		va, vb := stack.Get(1), stack.Get(2)
		assert.Equal(t, va, vb)
		assert.Equal(t, 23, int(va.(cpi.KNumber)))
	})

	t.Run("move_list", func(t *testing.T) {
		src := `
		var a = [] 
		var b = a 
		append(b, 1) 
		append(a, "kmp")
		`
		proto := compile(src)
		state := Prepare(proto)
		state.Run(proto.InstList.LastIndex())
		stack := state.stackValue
		va := stack.Get(1)
		vb := stack.Get(2)

		assert.Equal(t, va.Type(), cpi.KTypeList)
		assert.Equal(t, va, vb)
		assert.Equal(t, 2, va.(cpi.KList).Len())
		assert.Equal(t, 1, int(vb.(cpi.KList).GetAt(0).(cpi.KNumber)))
		assert.Equal(t, "kmp", string(vb.(cpi.KList).GetAt(1).(cpi.KString)))
	})

	t.Run("move_dict", func(t *testing.T) {
		src := `
		var a = {}
		var b 
		b = a 
		a.d, b.c = 1, "kmp"  
		`
		proto := compile(src)
		state := Prepare(proto)
		state.Run(proto.InstList.LastIndex())
		stack := state.stackValue
		va := stack.Get(1)
		vb := stack.Get(2)
		assert.Equal(t, va, vb)
		assert.Equal(t, 1, int(vb.(cpi.KDict).GetField("d").(cpi.KNumber)))
		assert.Equal(t, "kmp", string(va.(cpi.KDict).GetField("c").(cpi.KString)))

	})
}

func TestArithmetic(t *testing.T) {
	src := `
		var a, b = 10, 23
		a = b - 23 + 10 *a
		var c = a/5/4 
		var r = c % 3 
	`

	proto := compile(src)
	state := Prepare(proto)
	state.Run(proto.InstList.LastIndex())
	stack := state.stackValue
	va, vb := stack.Get(1), stack.Get(2)
	vc, vr := stack.Get(3), stack.Get(4)

	assert.Equal(t, 100, int(va.(cpi.KNumber)))
	assert.Equal(t, 23, int(vb.(cpi.KNumber)))
	assert.Equal(t, 5, int(vc.(cpi.KNumber)))
	assert.Equal(t, 2, int(vr.(cpi.KNumber)))
}

func TestFunctionCall(t *testing.T) {
	t.Run("one_return_value", func(t *testing.T) {
		src := `
		func max(x,y) {
			if x > y {
				return x
			}
			return y
		}
		var m,n = 102, 22
		var a = max(n,m)		
	`
		proto := compile(src)
		state := Prepare(proto)
		state.Run(proto.InstList.LastIndex())
		stack := state.stackValue
		fn := stack.Get(1).(*ClosureFunc)
		assert.Equal(t, 2, fn.Proto.NumParams)
		a := stack.Get(4).(cpi.KNumber)
		assert.Equal(t, 102, int(a))
	})
	t.Run("multi_return_value", func(t *testing.T) {
		src := `
		func sort(x,y) {
			if x > y {
				return y,x
			}
			return x,y
		}
		var m,n = 102, 22
		var a,b 
		a,b = sort(m,n)			
	`
		proto := compile(src)
		state := Prepare(proto)
		state.Run(proto.InstList.LastIndex())
		stack := state.stackValue
		a, b := stack.Get(4), stack.Get(5)

		assert.Equal(t, 22, int(a.(cpi.KNumber)))
		assert.Equal(t, 102, int(b.(cpi.KNumber)))
	})
	t.Run("use_upvalue_1", func(t *testing.T) {
		src := `
		var a = [] 
		func calc(x) {
			append(a,x)
		}
		calc(2)
		calc(3)
		`
		proto := compile(src)
		state := Prepare(proto)
		fmt.Println("last inst idx:", proto.InstList.LastIndex())
		state.Run(proto.InstList.LastIndex())
		stack := state.stackValue
		a := stack.Get(1).(cpi.KList)
		assert.Equal(t, 2, a.Len())
		assert.Equal(t, 2, int(a.GetAt(0).(cpi.KNumber)))
		assert.Equal(t, 3, int(a.GetAt(1).(cpi.KNumber)))
	})
	t.Run("use_upvalue_2", func(t *testing.T) {
		src := `
		var a = []
		func calc(x) {
			return func() {
				x = x*2
				append(a,x)
			}
		}
		var f2 = calc(2) 
		var f3 = calc(3) 
		f2()		
		f2()		
		`
		proto := compile(src)
		state := Prepare(proto)
		state.Run(proto.InstList.LastIndex())
		stack := state.stackValue
		a := stack.Get(1).(cpi.KList)
		assert.Equal(t, 2, a.Len())
		assert.Equal(t, 4, int(a.GetAt(0).(cpi.KNumber)))
		assert.Equal(t, 8, int(a.GetAt(1).(cpi.KNumber)))
		//assert.Equal(t, 3, int(a.GetAt(1).(cpi.KNumber)))
	})

	t.Run("param_dict", func(t *testing.T) {

		src := `
		var a={}
		func calc(d) {
			a.name = 'kmp'
			d.age = 1
			a.id = 1
		}
		calc({})	
		`
		proto := compile(src)
		state := Prepare(proto)
		state.Run(proto.InstList.LastIndex())
		stack := state.stackValue
		a := stack.Get(1).(cpi.KDict)
		assert.Equal(t, 1, int(a.GetField("id").(cpi.KNumber)))
		assert.Equal(t, "kmp", string(a.GetField("name").(cpi.KString)))
	})
}

func TestIf(t *testing.T) {
	t.Run("if_else", func(t *testing.T) {
		src := `
		var a,b,c = [2,1,2],0,0
		if a[2] == 2 {
			b=1
		} else {
			b=2
		}
		if a[1] > 1 {
			c = 1
		} else {
			c=2
		}
	`
		proto := compile(src)
		state := Prepare(proto)
		state.Run(proto.InstList.LastIndex())
		stack := state.stackValue
		a, b, c := stack.Get(1), stack.Get(2), stack.Get(3)

		assert.Equal(t, 3, a.(cpi.KList).Len())
		assert.Equal(t, 1, int(b.(cpi.KNumber)))
		assert.Equal(t, 2, int(c.(cpi.KNumber)))
	})
	t.Run("if_not_else", func(t *testing.T) {
		src := `
		var a,b,c = [2,1,2],0,0
		if a[2] != 2 {
			b=2
		}
		if a[1] > 1 {
			c = 1
		}
		`
		proto := compile(src)
		state := Prepare(proto)
		state.Run(proto.InstList.LastIndex())
		stack := state.stackValue
		b, c := stack.Get(2), stack.Get(3)
		assert.Equal(t, 0, int(b.(cpi.KNumber)))
		assert.Equal(t, 0, int(c.(cpi.KNumber)))
	})
	t.Run("else_if", func(t *testing.T) {
		src := `
		var a,b,c = [2,1,2],0,0
		if a[2] > 2 {
			b=2
		} else if a[2] == 2 {
			b=1
		}
		if a[1] > 1 {
			c = 1
		} else if a[1] < 1 {
			c= 3
		} else {
			c= 2
		}
		`
		proto := compile(src)
		state := Prepare(proto)
		state.Run(proto.InstList.LastIndex())
		stack := state.stackValue
		b, c := stack.Get(2), stack.Get(3)
		assert.Equal(t, 1, int(b.(cpi.KNumber)))
		assert.Equal(t, 2, int(c.(cpi.KNumber)))
	})
	t.Run("if_logical_exp", func(t *testing.T) {
		src := `
			var a = [2,1,3]
			var b, c,d,e =0,0,0
			if a[0] < 2 or a[1] == 1{
				b = 1
			} 
		
			if a[0] == 2 and a[2] == 3 {
				c = 1
			}
			
			if a[0] <2 or (a[2] == 3 and a[1]==1) {
				d = 1
			}
			if a[2] == 3 and (a[0] < 1 or a[1]>0) {
				e = 1
			}
		`
		proto := compile(src)
		state := Prepare(proto)
		state.Run(proto.InstList.LastIndex())
		stack := state.stackValue
		b, c, d, e := stack.Get(2), stack.Get(3), stack.Get(4), stack.Get(5)

		assert.Equal(t, 1, int(b.(cpi.KNumber)))
		assert.Equal(t, 1, int(c.(cpi.KNumber)))
		assert.Equal(t, 1, int(d.(cpi.KNumber)))
		assert.Equal(t, 1, int(e.(cpi.KNumber)))
	})
}

func TestWhile(t *testing.T) {
	src := `
		var a,cnt = 3, 0 
		while a > 0 {
			cnt = cnt + 1
			a = a-1
		}
	`
	proto := compile(src)
	state := Prepare(proto)
	state.Run(proto.InstList.LastIndex())
	stack := state.stackValue
	a, cnt := stack.Get(1), stack.Get(2)
	assert.Equal(t, 0, int(a.(cpi.KNumber)))
	assert.Equal(t, 3, int(cnt.(cpi.KNumber)))
}

func TestForNumber(t *testing.T) {
	t.Run("one_loop", func(t *testing.T) {
		src := `
		var a = {x: 1, y:0}
		for i=0,5 {
			a.x = a.x+1
		}	
		for i=1,10, 2 {
			a.y = a.y + i
		}
	`
		proto := compile(src)
		state := Prepare(proto)
		state.Run(proto.InstList.LastIndex())
		stack := state.stackValue
		a := stack.Get(1).(cpi.KDict)
		assert.Equal(t, 2, a.Len())
		assert.Equal(t, 6, int(a.GetField("x").(cpi.KNumber)))
		assert.Equal(t, 25, int(a.GetField("y").(cpi.KNumber)))
	})

	t.Run("nested_loop", func(t *testing.T) {
		src := `
			var a = []
			for i=1,6 {
				for j=i+1,6{
					append(a, [i,j])
				}
			}					
		`
		proto := compile(src)
		state := Prepare(proto)
		state.Run(proto.InstList.LastIndex())
		stack := state.stackValue
		a := stack.Get(1).(cpi.KList)
		assert.Equal(t, 10, a.Len())
	})
}
func TestRelation(t *testing.T) {
	src := `
		var a,b = 1<2, 12
		var c = b > 20
	`
	proto := compile(src)
	state := Prepare(proto)
	state.Run(proto.InstList.LastIndex())
	stack := state.stackValue
	a, c := stack.Get(1).(cpi.KBool), stack.Get(3).(cpi.KBool)
	assert.Equal(t, true, bool(a))
	assert.Equal(t, false, bool(c))
}

func TestString(t *testing.T) {
	src := `
		var a, b= "kmp", "123" 
		a = a..b 
		var c = b.."a"
	`
	proto := compile(src)
	state := Prepare(proto)
	state.Run(proto.InstList.LastIndex())
	stack := state.stackValue
	a, c := stack.Get(1).(cpi.KString), stack.Get(3).(cpi.KString)
	assert.Equal(t, "kmp123", string(a))
	assert.Equal(t, "123a", string(c))
}

func TestClosure(t *testing.T) {
	t.Run("closure_in_forLoop", func(t *testing.T) {
		src := `
		var cbs = [] 
		func initCbs() {
			for i=0,3 {
				var j = i 
				append(cbs, func() {
					j= j+1
					return j * j					
				})
			}
		}
		initCbs()
		var a,b,c = cbs[0](), cbs[1](),cbs[2]()		
		var d = cbs[0]()
	`

		proto := compile(src)
		state := Prepare(proto)
		state.Run(proto.InstList.LastIndex())
		stack := state.stackValue
		a, b, c := stack.Get(3), stack.Get(4), stack.Get(5)
		assert.Equal(t, 1, int(a.(cpi.KNumber)))
		assert.Equal(t, 4, int(b.(cpi.KNumber)))
		assert.Equal(t, 9, int(c.(cpi.KNumber)))
		d := stack.Get(6)
		assert.Equal(t, 4, int(d.(cpi.KNumber)))
	})

	t.Run("closure_in_while", func(t *testing.T) {
		src := `
			var cnt,lst = 1 ,[]
			while cnt < 5{
				var j = cnt 
				append(lst,func(){
					j = j * 2
					return j + 1 
				})
				cnt = cnt + 1
				if cnt % 3 == 0 {
					break
				}
			}
			var a, b = lst[0](),lst[1]() 
		`
		proto := compile(src)
		state := Prepare(proto)
		state.Run(proto.InstList.LastIndex())
		stack := state.stackValue
		lst := stack.Get(2).(cpi.KList)
		a, b := stack.Get(3).(cpi.KNumber), stack.Get(4).(cpi.KNumber)
		assert.Equal(t, 3, int(a))
		assert.Equal(t, 5, int(b))
		assert.Equal(t, 2, lst.Len())
	})
}
func TestBreak(t *testing.T) {
	src := `
		var a = 12 
		while true {
			a = a - 1
			if a == 3 {break}
		} 
	`
	proto := compile(src)
	state := Prepare(proto)
	state.Run(proto.InstList.LastIndex())
	stack := state.stackValue
	a := stack.Get(1).(cpi.KNumber)
	assert.Equal(t, 3, int(a))
}

func TestDict(t *testing.T) {
	src := `
		var d = {}
		d.name = "a" 
		d.major ="cs"
		d.id = 1
		d.calc = func() {
			return d.name .. "-" .. d.major
		}
		var name,id = d.name,d.id
		var a = d.calc()
	`

	proto := compile(src)
	state := Prepare(proto)
	state.Run(proto.InstList.LastIndex())
	stack := state.stackValue
	name, id := stack.Get(2).(cpi.KString), stack.Get(3).(cpi.KNumber)
	a := stack.Get(4).(cpi.KString)

	assert.Equal(t, "a", string(name))
	assert.Equal(t, 1, int(id))
	assert.Equal(t, "a-cs", string(a))
}
func TestUnary(t *testing.T) {
	src := `
		var a, b = 10,22	
		var d,e = !(a < b), -a
	`
	proto := compile(src)
	state := Prepare(proto)
	state.Run(proto.InstList.LastIndex())
	stack := state.stackValue
	d, e := stack.Get(3).(cpi.KBool), stack.Get(4).(cpi.KNumber)
	assert.Equal(t, false, bool(d))
	assert.Equal(t, -10, int(e))
}
func TestLogialOp(t *testing.T) {
	src := `
	var a , b = 12, 23
	var c = (a < 10) or b > 30 or b < 25
	var d = a < 20
	`
	proto := compile(src)
	state := Prepare(proto)
	state.Run(proto.InstList.LastIndex())
	stack := state.stackValue
	//stack.PrintRange(0, -1)
	c := stack.Get(3).(cpi.KBool)
	d := stack.Get(4).(cpi.KBool)
	assert.Equal(t, true, bool(c))
	assert.Equal(t, true, bool(d))
}

func TestForRange(t *testing.T) {
	t.Run("list", func(t *testing.T) {
		src := `
		var a = [2,3,1] 
		var s = 0 
		for i,v = range a {
			a[i] = a[i] * 2 +1
			s = s + a[i]
		}
	`
		proto := compile(src)
		state := Prepare(proto)
		state.Run(proto.InstList.LastIndex())
		stack := state.stackValue
		s := stack.Get(2).(cpi.KNumber)
		assert.Equal(t, 15, int(s))
	})

	t.Run("dict", func(t *testing.T) {
		src := `
			var a = {name:'kmp', id:1}
			var lst = []									
			for k, v = range a {
				append(lst, v)
			}	
		`
		proto := compile(src)
		state := Prepare(proto)
		state.Run(proto.InstList.LastIndex())
		stack := state.stackValue
		lst := stack.Get(2).(cpi.KList)
		assert.Equal(t, 2, lst.Len())
		assert.Equal(t, "kmp", string(lst.GetAt(0).(cpi.KString)))
		assert.Equal(t, 1, int(lst.GetAt(1).(cpi.KNumber)))
	})
}
