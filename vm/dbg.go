package vm

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strconv"

	"github.com/khoakmp/kala/cpi"
)

func (s *StackValue) PrintRange(from, to int) {
	if to == -1 || to > s.top {
		to = s.top
	}

	for i := to - 1; i >= from; i-- {
		v := s.Get(i)
		if v == nil {
			fmt.Printf("Reg[%d], Type: nil\n", i)
			continue
		}
		fmt.Printf("Reg[%d], Type: %s, Value: %s\n", i, cpi.TypeNames[v.Type()], v.Str())
	}
}

func RunStepByStep(proto *cpi.FuncProto) {
	state := Prepare(proto)
	rd := bufio.NewReader(os.Stdin)
	for {
		cmd, _, err := rd.ReadLine()
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		if len(cmd) == 0 {
			if state.currentFrame == nil {
				fmt.Println("DONE")
				return
			}
			step(state)
			continue
		}
		if bytes.Equal(cmd, []byte("q")) {
			return
		}
		lst := bytes.Split(cmd, []byte(" "))

		switch {
		case bytes.Equal(lst[0], []byte("pst")):
			from, err := strconv.Atoi(string(lst[1]))
			if err != nil {
				fmt.Println(err)
				continue
			}
			to, err := strconv.Atoi(string(lst[2]))
			if err != nil {
				fmt.Println(err)
				continue
			}
			state.stackValue.PrintRange(from, to)
		case bytes.Equal(lst[0], []byte("pkst")):
			state.currentFrame.Closure.Proto.Consts.Print()
		}
	}
}

func step(s *RuntimeState) {
	cf := s.currentFrame

	inst := cf.Closure.Proto.InstList.At(cf.PC)
	fmt.Println(cpi.InstToString(inst))
	cf.PC++
	execFunc[opGetOpCode(inst)](s, inst)
}
