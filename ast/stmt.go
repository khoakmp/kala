package ast

type Stmt interface{}

type IfStmt struct {
	CondExpr  Expr
	ThenChunk []Stmt
	ElseChunk []Stmt
}

type AssignStmt struct {
	Lhs []Expr
	Rhs []Expr
}

type WhileStmt struct {
	CondExpr Expr
	Chunk    []Stmt
}

type ForNumberStmt struct {
	CounterName      string
	Start, End, Step Expr
	Chunk            []Stmt
}

type ReturnStmt struct {
	Exprs []Expr
}

type BreakStmt struct{}

type FuncDefStmt struct {
	FuncName string
	ParList  []string
	Block    []Stmt
	HasVArg  bool
}

type VarDefStmt struct {
	Vars  []string
	Exprs []Expr
}
type ParList struct {
	Names   []string
	HasVArg bool
}

type FuncCallStmt struct {
	Expr *FuncCallExpr
}

type ListAppendStmt struct {
	Object  Expr
	Element Expr
}

type ForRangeStmt struct {
	Index  string
	Value  string
	Object Expr
	Block  []Stmt
}
