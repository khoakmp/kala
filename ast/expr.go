package ast

const (
	OpAnd = iota
	OpOr
	OpXor
	OpAdd
	OpSubtract
	OpMul
	OpDiv
	OpMod
	OpBitOr
	OpBitAnd
	OpNot
	OpMinus
	OpLt
	OpLe
	OpGt
	OpGe
	OpEqual
	OpNotEqual
)

type Expr interface{}

type StringExpr struct {
	Value string
}

type NumberExpr struct {
	Value string
}

type NilExpr struct{}
type TrueExpr struct{}
type FalseExpr struct{}

type IdentExpr struct {
	Value string
}

type LogicalOpExpr struct {
	Operator int // And, Or, Xor,
	Lhs, Rhs Expr
}

type RelationalOpExpr struct {
	Operator int // >,<,>=, <=, ==, !=
	Lhs, Rhs Expr
}

type ArithmeticOpExpr struct {
	Operator int // Add,Sub,Mul,Div,Mod
	Lhs, Rhs Expr
}

type UnaryOpNotExpr struct {
	Expr Expr
}
type UnaryOpMinusExpr struct {
	Expr Expr
}
type FieldGetExpr struct {
	Object Expr
	Key    Expr
}

type FunctionExpr struct {
	Params  []string
	HasVArg bool
	Block   []Stmt
}

type FuncCallExpr struct {
	Func Expr
	Args []Expr
}

type ConcatStrExpr struct {
	Lhs Expr
	Rhs Expr
}

type DictExpr struct {
	Entries []DictEntry
}

type DictEntry struct {
	Key   string
	Value Expr
}

type ListExpr struct {
	Elements []Expr
}

type LenExpr struct {
	Object Expr
}
