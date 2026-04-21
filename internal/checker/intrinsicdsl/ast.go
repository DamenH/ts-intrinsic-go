package intrinsicdsl

// Node is a tagged union for all DSL AST nodes.
type Node struct {
	Kind NodeKind

	NumVal  float64
	StrVal  string
	BoolVal bool

	Left     *Node
	Right    *Node
	Op       string
	Args     []*Node
	Body     *Node
	Params   []string
	Callee   *Node
	Prop     string
	Cond     *Node
	Then     *Node
	Else     *Node
	ObjProps []ObjProp
	ArrElems []ArrElem
	Stmts    []Stmt
	Preamble []Stmt // let declarations injected before the main function (for dependencies)
}

type NodeKind int

const (
	NodeNumberLit NodeKind = iota
	NodeStringLit
	NodeBooleanLit
	NodeNullLit
	NodeUndefinedLit
	NodeIdent
	NodeBinary
	NodeUnary
	NodeTernary
	NodePropAccess
	NodeIndexAccess
	NodeCall
	NodeObjectLit
	NodeArrayLit
	NodeLambda
	NodeBlock
	NodeProgram
)

type ObjProp struct {
	Key    *Node
	Value  *Node
	Spread bool
}

type ArrElem struct {
	Expr   *Node
	Spread bool
}

type Stmt struct {
	Kind StmtKind

	Name    string
	Names   []string
	Rest    string
	Init    *Node
	Value   *Node
	Cond    *Node
	Then    []Stmt
	ElseIfs []ElseIf
	Else    []Stmt
	Object  *Node
	Index   *Node
	Iter    *Node
}

type StmtKind int

const (
	StmtLet StmtKind = iota
	StmtDestructureLet
	StmtAssign
	StmtIndexAssign
	StmtIf
	StmtForOf
	StmtWhile
	StmtBreak
	StmtContinue
	StmtReturn
	StmtExpr
)

type ElseIf struct {
	Cond *Node
	Body []Stmt
}
