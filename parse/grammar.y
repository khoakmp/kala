%{
package parse
import "github.com/khoakmp/kala/ast"
%}

%type<stmts> chunk chunk1 block
%type<stmt> laststmt stmt forNumStmt  ifstmt forRangeStmt
%type<exprlist> lhslist exprlist args
%type<expr> lhs prefixexp expr functioncall dictConstructor listConstructor
%type<namelist> namelist
%type<parlist> parlist
%type<entries> entries
%type<entry> entry

%union{
  token ast.Token
  stmt ast.Stmt 
  expr ast.Expr
  stmts []ast.Stmt
  exprlist []ast.Expr 
  namelist []string
  parlist *ast.ParList
  entries []ast.DictEntry
  entry ast.DictEntry
}

/* Reserved words */
%token<token> If Else For While Break Return And Or Function True False Nil Var Append Range


/* Literals , get Str of TNumber, TString, TIdent */
%token<token> Number String Ident Eq2 Neq Ge Le Dot3 Dot2 '{' '(' '!' '.'

/* Operators */
%left Or
%left And
%left '>' '<' Ge Le Eq2 Neq
%right Dot2
%left '+' '-'
%left '*' '/' '%'
%right UNARY /* not # -(unary) */
%right '^'

%%
  chunk: chunk1 {
    $$ = $1
    if l, ok:= yylex.(*Lexer); ok {
      l.Stmts = $$
    }
  } | chunk1 laststmt {
    $$ = append($1, $2)
    if l, ok:= yylex.(*Lexer); ok {
      l.Stmts = $$
    }    
  } | chunk1 laststmt ';'{
    $$ = append($1, $2) 
    if l,ok:= yylex.(*Lexer);ok {
      l.Stmts = $$
    }
  }
  
  chunk1: {
    $$ = []ast.Stmt{}  
  } | chunk1 stmt {
    $$ = append($1, $2)
  } | chunk1 ';' {
    $$ = $1
  } 
  
  laststmt: Break {
    $$ = &ast.BreakStmt{}
  } | Return {
    $$ = &ast.ReturnStmt{Exprs: []ast.Expr{}}
  } | Return exprlist {
    $$ = &ast.ReturnStmt{Exprs:  $2}    
  }
  
  stmt: lhslist '=' exprlist{
    $$ = &ast.AssignStmt {Lhs: $1, Rhs: $3}
  } | While expr block{
    $$ = &ast.WhileStmt {CondExpr: $2, Chunk: $3}
  } | ifstmt {
    $$ = $1
  } | forNumStmt {
    $$ = $1 
  } | forRangeStmt{
    $$ = $1
  } | Function Ident parlist block {
    $$ = &ast.FuncDefStmt {FuncName: $2.Str, ParList: $3.Names, HasVArg: $3.HasVArg, Block: $4}
  } | Var namelist {
    $$ = &ast.VarDefStmt{Vars : $2, Exprs : []ast.Expr{} }
  } | Var namelist '=' exprlist {
    $$ = &ast.VarDefStmt {Vars: $2, Exprs: $4}
  } | functioncall {
    if e , ok:= $1.(*ast.FuncCallExpr); ok {
      $$ = &ast.FuncCallStmt{
        Expr: e,
      }  
    } else {
      yylex.(*Lexer).Error("parse error")
    }
  } | Append '(' lhs ',' expr ')' {
    $$ = &ast.ListAppendStmt{
      Object: $3, 
      Element: $5,
    }
  }
  
  ifstmt: If  expr  block {
    $$ = &ast.IfStmt{CondExpr: $2, ThenChunk: $3, ElseChunk: []ast.Stmt{}}  
  } | If expr block Else block {
    $$ = &ast.IfStmt{CondExpr: $2, ThenChunk: $3, ElseChunk: $5}
  } | If expr block Else ifstmt {
    $$ = &ast.IfStmt{CondExpr: $2, ThenChunk: $3, ElseChunk: []ast.Stmt{$5}}
  }
  
  forRangeStmt: For Ident ',' Ident '=' Range lhs block {
    $$ = &ast.ForRangeStmt{
      Index: $2.Str,
      Value: $4.Str,
      Object: $7,
      Block: $8,
    }
  }
  forNumStmt: For Ident '=' expr ',' expr  block {
    $$ = &ast.ForNumberStmt { CounterName: $2.Str, Start: $4, End: $6, Step: nil, Chunk: $7}
  }  | For Ident '=' expr ',' expr ',' expr block {
    $$ = &ast.ForNumberStmt { CounterName: $2.Str, Start: $4, End: $6, Step: $8, Chunk: $9}
  }
  
  parlist: '(' ')'{
    $$ = &ast.ParList{Names :[]string{}, HasVArg: false}
  } | '(' namelist ')' {
    $$ = &ast.ParList {Names : $2 , HasVArg: false}
  }| '(' namelist ',' Dot3 ')' {
    $$ = &ast.ParList{Names: $2, HasVArg: true}
  }
  
  namelist: Ident{
    $$ = []string{$1.Str}
  } | namelist ',' Ident {
    $$ = append($1, $3.Str)
  }

  block: '{' chunk '}' {
    $$ = $2
  }

  lhslist: lhs {
    $$ = []ast.Expr{$1}    
  } | lhslist ',' lhs {
    $$ = append($1, $3)
  }

  exprlist: expr{
    $$ = []ast.Expr{$1}
  } | exprlist ',' expr {
    $$ = append($1, $3)
  }
  
  lhs: Ident {
    $$ = &ast.IdentExpr {Value: $1.Str}
  } | prefixexp '.' Ident {
    $$ = &ast.FieldGetExpr{Object: $1, Key: &ast.StringExpr{Value: $3.Str}}
  } | prefixexp '[' expr ']' {
    $$ = &ast.FieldGetExpr{Object: $1, Key: $3}
  } 
  
  prefixexp: lhs {
    $$ = $1
  } | functioncall {
    $$ = $1
  }

  functioncall: prefixexp '(' ')' {
    $$ = &ast.FuncCallExpr{Func: $1, Args :[]ast.Expr{}}
  } | prefixexp '(' args ')'{
    $$ = &ast.FuncCallExpr{Func: $1, Args: $3}    
  }
  
  args: expr {
    $$ = []ast.Expr{$1}
  } | args ',' expr {
    $$ = append($1, $3)
  } 
  
  expr: True {
    $$ = &ast.TrueExpr{}
  } | False {
    $$ = &ast.FalseExpr{}
  } | Nil {
    $$ = &ast.NilExpr{}
  } | Number {
    $$ = &ast.NumberExpr{Value: $1.Str}
  } | String {
    $$ = &ast.StringExpr{Value: $1.Str} 
  } | prefixexp {
    $$ = $1
  } | Function parlist block {
    $$ = &ast.FunctionExpr{
      Params: $2.Names, 
      HasVArg: $2.HasVArg,
      Block: $3,
    }
  } | expr '+' expr {
    $$ = &ast.ArithmeticOpExpr{
      Operator: ast.OpAdd,
      Lhs: $1, Rhs: $3,
    }
  } | expr '-' expr {
    $$ = &ast.ArithmeticOpExpr{
      Operator: ast.OpSubtract,
      Lhs: $1, Rhs: $3,
    }
  } | expr '*' expr {
    $$ = &ast.ArithmeticOpExpr{
      Operator: ast.OpMul,
      Lhs: $1, Rhs: $3,
    }
  } | expr '/' expr {
    $$ = &ast.ArithmeticOpExpr{
      Operator: ast.OpDiv,
      Lhs: $1, Rhs: $3,
    } 
  } | expr '%' expr {
    $$ = &ast.ArithmeticOpExpr{
      Operator: ast.OpMod,
      Lhs: $1, Rhs: $3,
    }
  } | expr '|' expr {
    $$ = &ast.ArithmeticOpExpr{
      Operator: ast.OpBitOr,
      Lhs: $1, Rhs: $3,
    }
  } | expr '&' expr {
    $$ = &ast.ArithmeticOpExpr{ Operator: ast.OpBitAnd, Lhs: $1, Rhs: $3}
  } | expr And expr {
    $$ = &ast.LogicalOpExpr { Operator: ast.OpAnd, Lhs: $1, Rhs: $3}
  } | expr Or expr {
    $$ = &ast.LogicalOpExpr { Operator: ast.OpOr, Lhs: $1, Rhs: $3}
  } | expr '<' expr {
    $$ = &ast.RelationalOpExpr { Operator: ast.OpLt, Lhs: $1, Rhs: $3}
  } | expr '>' expr {
    $$ = &ast.RelationalOpExpr { Operator: ast.OpGt, Lhs: $1, Rhs: $3}
  } | expr Le expr {
    $$ = &ast.RelationalOpExpr { Operator: ast.OpLe, Lhs: $1, Rhs: $3}
  } | expr Ge expr {
    $$ = &ast.RelationalOpExpr { Operator: ast.OpGe, Lhs: $1, Rhs: $3}
  } | expr Eq2 expr {
    $$ = &ast.RelationalOpExpr { Operator: ast.OpEqual, Lhs: $1, Rhs: $3}
  } | expr Neq expr {
    $$ = &ast.RelationalOpExpr { Operator: ast.OpNotEqual, Lhs: $1, Rhs: $3}
  } | '(' expr ')' {
    $$ = $2
  } | expr Dot2 expr {
    $$ = &ast.ConcatStrExpr {Lhs: $1, Rhs: $3}
  } | '-' expr %prec UNARY {
    $$ = &ast.UnaryOpMinusExpr {Expr: $2}
  } | '!' expr %prec UNARY {
    $$ = &ast.UnaryOpNotExpr {Expr : $2}
  } | dictConstructor{
    $$ = $1
  } | listConstructor {
    $$ = $1
  } | '#' expr {
    $$ = &ast.LenExpr{
      Object: $2,
    }     
  }

  dictConstructor: '{' '}'{
    $$ = &ast.DictExpr {
      Entries: []ast.DictEntry{},
    }
    
  } | '{' entries '}' {
    $$ = &ast.DictExpr {
      Entries: $2,
    }
  }

  entries: entry {
    $$ = []ast.DictEntry{$1}
  } | entries ',' entry {
    $$ = append($1, $3)
  }
  
  entry: String ':' expr {
    $$ = ast.DictEntry{
      Key: $1.Str,
      Value: $3,
    }
  } | Ident ':' expr {
    $$ = ast.DictEntry{
      Key: $1.Str,
      Value: $3,
    }
  }

  listConstructor: '[' ']'{
    $$ = &ast.ListExpr{
      Elements: []ast.Expr{},
    }
  } | '[' exprlist ']' {
    $$ = &ast.ListExpr{
      Elements: $2,
    }
  }
%%

func TokenName(c int) string {
	if c >= And && c-And < len(yyToknames) {
		if yyToknames[c-And] != "" {
			return yyToknames[c-And]
		}
	}
    return string([]byte{byte(c)})
}