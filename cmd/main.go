package main

import (
	"fmt"
	"os"
	"time"

	"github.com/khoakmp/kala/cpi"
	"github.com/khoakmp/kala/parse"
	"github.com/khoakmp/kala/vm"
)

func compile() {
	f, _ := os.Open("f.txt")
	defer f.Close()
	chunk, err := parse.Parse(f, "s")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(len(chunk))

	proto := cpi.Compile(chunk)
	for _, inst := range proto.InstList.List() {
		fmt.Println(cpi.InstToString(inst))
	}
}
func runStepByStep() {
	f, _ := os.Open("f.txt")
	defer f.Close()
	chunk, err := parse.Parse(f, "s")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(len(chunk))

	proto := cpi.Compile(chunk)
	fmt.Println("last inst idx:", proto.InstList.LastIndex())
	vm.RunStepByStep(proto)
}

func run() {
	f, _ := os.Open("f.txt")
	defer f.Close()
	st := time.Now()
	chunk, err := parse.Parse(f, "s")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("parse time:", time.Since(st))

	fmt.Println(len(chunk))
	st = time.Now()
	proto := cpi.Compile(chunk)
	fmt.Println("compile time:", time.Since(st))
	st = time.Now()
	vm.Run(proto)
	fmt.Println("run time:", time.Since(st))
}
func main() {
	args := os.Args
	if len(args) == 1 {
		run()
		return
	}

	switch args[1] {
	case "s":
		runStepByStep()
	case "c":
		compile()
	}
}
