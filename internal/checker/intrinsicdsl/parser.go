package intrinsicdsl

import "fmt"

func ParseProgram(src string) (*Node, error) {
	tokens, err := Lex(src)
	if err != nil {
		return nil, err
	}
	p := &parser{tokens: tokens}
	node, err := p.parseProgram()
	if err != nil {
		return nil, err
	}
	if !p.atEnd() {
		return nil, fmt.Errorf("unexpected token after program at position %d", p.peek().Pos)
	}
	return node, nil
}

type parser struct {
	tokens []Token
	pos    int
}

func (p *parser) peek() Token {
	if p.pos >= len(p.tokens) {
		return Token{Kind: TokEOF}
	}
	return p.tokens[p.pos]
}

func (p *parser) advance() Token {
	tok := p.peek()
	if p.pos < len(p.tokens) {
		p.pos++
	}
	return tok
}

func (p *parser) expect(kind TokenKind) (Token, error) {
	tok := p.peek()
	if tok.Kind != kind {
		return tok, fmt.Errorf("expected token kind %d but got %d at position %d", kind, tok.Kind, tok.Pos)
	}
	p.advance()
	return tok, nil
}

func (p *parser) atEnd() bool {
	return p.peek().Kind == TokEOF
}

func (p *parser) check(kind TokenKind) bool {
	return p.peek().Kind == kind
}

func (p *parser) match(kind TokenKind) bool {
	if p.check(kind) {
		p.advance()
		return true
	}
	return false
}

func (p *parser) parseProgram() (*Node, error) {
	// Optional preamble: "let name = ...;" declarations before the main function.
	var preamble []Stmt
	for p.check(TokIdent) && p.peek().Str == "let" {
		stmt, err := p.parseStmt()
		if err != nil {
			return nil, err
		}
		preamble = append(preamble, stmt)
		p.match(TokSemi) // consume optional semicolons between preamble and main function
	}

	params, err := p.parseParamList()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(TokArrow); err != nil {
		return nil, fmt.Errorf("expected '=>' in program")
	}
	body, err := p.parseBodyOrExpr()
	if err != nil {
		return nil, err
	}
	node := &Node{Kind: NodeProgram, Params: params, Body: body}
	node.Preamble = preamble
	return node, nil
}

func (p *parser) parseParamList() ([]string, error) {
	if _, err := p.expect(TokLParen); err != nil {
		return nil, fmt.Errorf("expected '(' for parameter list")
	}
	var params []string
	for !p.check(TokRParen) && !p.atEnd() {
		name, err := p.expectIdent()
		if err != nil {
			return nil, err
		}
		params = append(params, name)
		p.skipTypeAnnotation()
		if !p.match(TokComma) {
			break
		}
	}
	if _, err := p.expect(TokRParen); err != nil {
		return nil, fmt.Errorf("expected ')' after parameters")
	}
	return params, nil
}

// skipTypeAnnotation consumes ": Type" tokens after a parameter name.
func (p *parser) skipTypeAnnotation() {
	if !p.match(TokColon) {
		return
	}
	depth := 0
	for !p.atEnd() {
		switch {
		case (p.check(TokComma) || p.check(TokRParen) || p.check(TokArrow) || p.check(TokAssign)) && depth == 0:
			return
		case p.check(TokLt) || p.check(TokLBrack) || p.check(TokLParen):
			depth++
			p.advance()
		case p.check(TokGt) || p.check(TokRBrack) || p.check(TokRParen):
			if depth == 0 {
				return
			}
			depth--
			p.advance()
		default:
			p.advance()
		}
	}
}

func (p *parser) expectIdent() (string, error) {
	tok := p.peek()
	if tok.Kind != TokIdent {
		return "", fmt.Errorf("expected identifier at position %d", tok.Pos)
	}
	p.advance()
	return tok.Str, nil
}

func (p *parser) parseBodyOrExpr() (*Node, error) {
	if p.check(TokLBrace) {
		return p.parseBlock()
	}
	return p.parseExpr()
}

func (p *parser) parseBlock() (*Node, error) {
	p.advance() // consume '{'
	stmts, err := p.parseStmts()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(TokRBrace); err != nil {
		return nil, fmt.Errorf("expected '}' to close block")
	}
	return &Node{Kind: NodeBlock, Stmts: stmts}, nil
}

func (p *parser) parseStmts() ([]Stmt, error) {
	var stmts []Stmt
	for !p.check(TokRBrace) && !p.atEnd() {
		stmt, err := p.parseStmt()
		if err != nil {
			return nil, err
		}
		stmts = append(stmts, stmt)
		p.match(TokSemi) // optional semicolons
	}
	return stmts, nil
}

func (p *parser) parseStmt() (Stmt, error) {
	tok := p.peek()

	if tok.Kind == TokIdent {
		switch tok.Str {
		case "let":
			return p.parseLetStmt()
		case "if":
			return p.parseIfStmt()
		case "for":
			return p.parseForOfStmt()
		case "while":
			return p.parseWhileStmt()
		case "break":
			p.advance()
			p.match(TokSemi)
			return Stmt{Kind: StmtBreak}, nil
		case "continue":
			p.advance()
			p.match(TokSemi)
			return Stmt{Kind: StmtContinue}, nil
		case "return":
			return p.parseReturnStmt()
		}
	}

	expr, err := p.parseExpr()
	if err != nil {
		return Stmt{}, err
	}

	if p.match(TokAssign) {
		val, err := p.parseExpr()
		if err != nil {
			return Stmt{}, err
		}
		if expr.Kind == NodeIdent {
			return Stmt{Kind: StmtAssign, Name: expr.StrVal, Value: val}, nil
		}
		if expr.Kind == NodeIndexAccess {
			return Stmt{Kind: StmtIndexAssign, Object: expr.Left, Index: expr.Right, Value: val}, nil
		}
		if expr.Kind == NodePropAccess {
			keyNode := &Node{Kind: NodeStringLit, StrVal: expr.Prop}
			return Stmt{Kind: StmtIndexAssign, Object: expr.Left, Index: keyNode, Value: val}, nil
		}
		return Stmt{}, fmt.Errorf("invalid assignment target at position %d", tok.Pos)
	}

	return Stmt{Kind: StmtExpr, Value: expr}, nil
}

func (p *parser) parseLetStmt() (Stmt, error) {
	p.advance() // consume "let"

	if p.check(TokLBrack) {
		return p.parseDestructureLet()
	}

	name, err := p.expectIdent()
	if err != nil {
		return Stmt{}, fmt.Errorf("expected variable name after 'let'")
	}
	p.skipTypeAnnotation()
	if _, err := p.expect(TokAssign); err != nil {
		return Stmt{}, fmt.Errorf("expected '=' after variable name in let")
	}
	init, err := p.parseExpr()
	if err != nil {
		return Stmt{}, err
	}
	return Stmt{Kind: StmtLet, Name: name, Init: init}, nil
}

func (p *parser) parseDestructureLet() (Stmt, error) {
	p.advance() // consume "["
	var names []string
	var rest string
	for !p.check(TokRBrack) && !p.atEnd() {
		if p.match(TokSpread) {
			r, err := p.expectIdent()
			if err != nil {
				return Stmt{}, fmt.Errorf("expected identifier after '...' in destructuring")
			}
			rest = r
			break
		}
		name, err := p.expectIdent()
		if err != nil {
			return Stmt{}, fmt.Errorf("expected identifier in destructuring")
		}
		names = append(names, name)
		if !p.match(TokComma) {
			break
		}
	}
	if _, err := p.expect(TokRBrack); err != nil {
		return Stmt{}, fmt.Errorf("expected ']' after destructuring names")
	}
	if _, err := p.expect(TokAssign); err != nil {
		return Stmt{}, fmt.Errorf("expected '=' after destructuring pattern")
	}
	init, err := p.parseExpr()
	if err != nil {
		return Stmt{}, err
	}
	return Stmt{Kind: StmtDestructureLet, Names: names, Rest: rest, Init: init}, nil
}

func (p *parser) parseIfStmt() (Stmt, error) {
	p.advance() // consume "if"
	if _, err := p.expect(TokLParen); err != nil {
		return Stmt{}, fmt.Errorf("expected '(' after 'if'")
	}
	cond, err := p.parseExpr()
	if err != nil {
		return Stmt{}, err
	}
	if _, err := p.expect(TokRParen); err != nil {
		return Stmt{}, fmt.Errorf("expected ')' after if condition")
	}
	then, err := p.parseStmtBlock()
	if err != nil {
		return Stmt{}, err
	}

	var elseIfs []ElseIf
	var elseBody []Stmt

	for p.check(TokIdent) && p.peek().Str == "else" {
		p.advance() // consume "else"
		if p.check(TokIdent) && p.peek().Str == "if" {
			p.advance() // consume "if"
			if _, err := p.expect(TokLParen); err != nil {
				return Stmt{}, fmt.Errorf("expected '(' after 'else if'")
			}
			eic, err := p.parseExpr()
			if err != nil {
				return Stmt{}, err
			}
			if _, err := p.expect(TokRParen); err != nil {
				return Stmt{}, fmt.Errorf("expected ')' after else-if condition")
			}
			eib, err := p.parseStmtBlock()
			if err != nil {
				return Stmt{}, err
			}
			elseIfs = append(elseIfs, ElseIf{Cond: eic, Body: eib})
		} else {
			eb, err := p.parseStmtBlock()
			if err != nil {
				return Stmt{}, err
			}
			elseBody = eb
			break
		}
	}

	return Stmt{Kind: StmtIf, Cond: cond, Then: then, ElseIfs: elseIfs, Else: elseBody}, nil
}

func (p *parser) parseStmtBlock() ([]Stmt, error) {
	if p.check(TokLBrace) {
		p.advance() // consume "{"
		stmts, err := p.parseStmts()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TokRBrace); err != nil {
			return nil, fmt.Errorf("expected '}' after block")
		}
		return stmts, nil
	}
	stmt, err := p.parseStmt()
	if err != nil {
		return nil, err
	}
	return []Stmt{stmt}, nil
}

func (p *parser) parseForOfStmt() (Stmt, error) {
	p.advance() // consume "for"
	if _, err := p.expect(TokLParen); err != nil {
		return Stmt{}, fmt.Errorf("expected '(' after 'for'")
	}
	if tok := p.peek(); tok.Kind != TokIdent || tok.Str != "let" {
		return Stmt{}, fmt.Errorf("expected 'let' in for-of")
	}
	p.advance() // consume "let"
	name, err := p.expectIdent()
	if err != nil {
		return Stmt{}, fmt.Errorf("expected binding name in for-of")
	}
	p.skipTypeAnnotation()
	if tok := p.peek(); tok.Kind != TokIdent || tok.Str != "of" {
		return Stmt{}, fmt.Errorf("expected 'of' in for-of")
	}
	p.advance() // consume "of"
	iter, err := p.parseExpr()
	if err != nil {
		return Stmt{}, err
	}
	if _, err := p.expect(TokRParen); err != nil {
		return Stmt{}, fmt.Errorf("expected ')' after for-of")
	}
	if _, err := p.expect(TokLBrace); err != nil {
		return Stmt{}, fmt.Errorf("expected '{' after for-of header")
	}
	body, err := p.parseStmts()
	if err != nil {
		return Stmt{}, err
	}
	if _, err := p.expect(TokRBrace); err != nil {
		return Stmt{}, fmt.Errorf("expected '}' after for-of body")
	}
	return Stmt{Kind: StmtForOf, Name: name, Iter: iter, Then: body}, nil
}

func (p *parser) parseWhileStmt() (Stmt, error) {
	p.advance() // consume "while"
	if _, err := p.expect(TokLParen); err != nil {
		return Stmt{}, fmt.Errorf("expected '(' after 'while'")
	}
	cond, err := p.parseExpr()
	if err != nil {
		return Stmt{}, err
	}
	if _, err := p.expect(TokRParen); err != nil {
		return Stmt{}, fmt.Errorf("expected ')' after while condition")
	}
	if _, err := p.expect(TokLBrace); err != nil {
		return Stmt{}, fmt.Errorf("expected '{' after while condition")
	}
	body, err := p.parseStmts()
	if err != nil {
		return Stmt{}, err
	}
	if _, err := p.expect(TokRBrace); err != nil {
		return Stmt{}, fmt.Errorf("expected '}' after while body")
	}
	return Stmt{Kind: StmtWhile, Cond: cond, Then: body}, nil
}

func (p *parser) parseReturnStmt() (Stmt, error) {
	p.advance() // consume "return"
	if p.check(TokSemi) || p.check(TokRBrace) {
		return Stmt{Kind: StmtReturn, Value: &Node{Kind: NodeUndefinedLit}}, nil
	}
	val, err := p.parseExpr()
	if err != nil {
		return Stmt{}, err
	}
	return Stmt{Kind: StmtReturn, Value: val}, nil
}

var binaryPrecTable = []map[TokenKind]string{
	{TokOr: "||"},
	{TokAnd: "&&"},
	{TokEq: "==", TokNEq: "!="},
	{TokLt: "<", TokGt: ">", TokLEq: "<=", TokGEq: ">="},
	{TokPlus: "+", TokMinus: "-"},
	{TokStar: "*", TokSlash: "/", TokPercent: "%"},
}

func (p *parser) parseExpr() (*Node, error) {
	return p.parseTernary()
}

func (p *parser) parseTernary() (*Node, error) {
	node, err := p.parseBinaryExpr(0)
	if err != nil {
		return nil, err
	}
	if p.match(TokQuestion) {
		then, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TokColon); err != nil {
			return nil, fmt.Errorf("expected ':' in ternary")
		}
		els, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		return &Node{Kind: NodeTernary, Cond: node, Then: then, Else: els}, nil
	}
	return node, nil
}

func (p *parser) parseBinaryExpr(level int) (*Node, error) {
	if level >= len(binaryPrecTable) {
		return p.parseUnary()
	}
	left, err := p.parseBinaryExpr(level + 1)
	if err != nil {
		return nil, err
	}
	for {
		opStr, ok := binaryPrecTable[level][p.peek().Kind]
		if !ok {
			break
		}
		p.advance()
		right, err := p.parseBinaryExpr(level + 1)
		if err != nil {
			return nil, err
		}
		left = &Node{Kind: NodeBinary, Op: opStr, Left: left, Right: right}
	}
	return left, nil
}

func (p *parser) parseUnary() (*Node, error) {
	if p.check(TokMinus) {
		p.advance()
		operand, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &Node{Kind: NodeUnary, Op: "-", Left: operand}, nil
	}
	if p.check(TokNot) {
		p.advance()
		operand, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &Node{Kind: NodeUnary, Op: "!", Left: operand}, nil
	}
	if p.check(TokIdent) && p.peek().Str == "void" {
		p.advance()
		operand, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &Node{Kind: NodeUnary, Op: "void", Left: operand}, nil
	}
	if p.check(TokIdent) && p.peek().Str == "typeof" {
		p.advance()
		operand, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &Node{Kind: NodeUnary, Op: "typeof", Left: operand}, nil
	}
	return p.parsePostfix()
}

func (p *parser) parsePostfix() (*Node, error) {
	node, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	for {
		if p.check(TokDot) {
			p.advance()
			name, err := p.expectIdent()
			if err != nil {
				return nil, fmt.Errorf("expected property name after '.'")
			}
			node = &Node{Kind: NodePropAccess, Left: node, Prop: name}
		} else if p.check(TokLBrack) {
			p.advance()
			index, err := p.parseExpr()
			if err != nil {
				return nil, err
			}
			if _, err := p.expect(TokRBrack); err != nil {
				return nil, fmt.Errorf("expected ']' after index")
			}
			node = &Node{Kind: NodeIndexAccess, Left: node, Right: index}
		} else if p.check(TokLParen) {
			args, err := p.parseCallArgs()
			if err != nil {
				return nil, err
			}
			node = &Node{Kind: NodeCall, Callee: node, Args: args}
		} else {
			break
		}
	}
	return node, nil
}

func (p *parser) parseCallArgs() ([]*Node, error) {
	p.advance() // consume "("
	var args []*Node
	for !p.check(TokRParen) && !p.atEnd() {
		arg, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
		if !p.match(TokComma) {
			break
		}
	}
	if _, err := p.expect(TokRParen); err != nil {
		return nil, fmt.Errorf("expected ')' after call arguments")
	}
	return args, nil
}

func (p *parser) parsePrimary() (*Node, error) {
	tok := p.peek()

	switch tok.Kind {
	case TokNum:
		p.advance()
		return &Node{Kind: NodeNumberLit, NumVal: tok.Num}, nil
	case TokStr:
		p.advance()
		return &Node{Kind: NodeStringLit, StrVal: tok.Str}, nil
	case TokLParen:
		return p.parseParenOrLambda()
	case TokLBrack:
		return p.parseArrayLit()
	case TokLBrace:
		return p.parseObjectLit()
	case TokIdent:
		return p.parseIdentOrKeyword()
	}

	return nil, fmt.Errorf("unexpected token at position %d", tok.Pos)
}

func (p *parser) parseIdentOrKeyword() (*Node, error) {
	tok := p.advance()
	switch tok.Str {
	case "true":
		return &Node{Kind: NodeBooleanLit, BoolVal: true}, nil
	case "false":
		return &Node{Kind: NodeBooleanLit, BoolVal: false}, nil
	case "null":
		return &Node{Kind: NodeNullLit}, nil
	case "undefined":
		return &Node{Kind: NodeUndefinedLit}, nil
	}
	return &Node{Kind: NodeIdent, StrVal: tok.Str}, nil
}

func (p *parser) parseParenOrLambda() (*Node, error) {
	saved := p.pos
	if params, ok := p.tryParseLambdaParams(); ok {
		if p.match(TokArrow) {
			body, err := p.parseBodyOrExpr()
			if err != nil {
				return nil, err
			}
			return &Node{Kind: NodeLambda, Params: params, Body: body}, nil
		}
	}

	p.pos = saved
	p.advance() // consume "("
	expr, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(TokRParen); err != nil {
		return nil, fmt.Errorf("expected ')' after expression")
	}
	return expr, nil
}

func (p *parser) tryParseLambdaParams() ([]string, bool) {
	p.advance() // consume "("

	if p.check(TokRParen) {
		p.advance()
		return nil, true
	}

	var params []string
	for {
		if !p.check(TokIdent) {
			return nil, false
		}
		params = append(params, p.advance().Str)
		p.skipTypeAnnotation()
		if p.check(TokRParen) {
			p.advance()
			return params, true
		}
		if !p.match(TokComma) {
			return nil, false
		}
	}
}

func (p *parser) parseArrayLit() (*Node, error) {
	p.advance() // consume "["
	var elems []ArrElem
	for !p.check(TokRBrack) && !p.atEnd() {
		spread := p.match(TokSpread)
		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		elems = append(elems, ArrElem{Expr: expr, Spread: spread})
		if !p.match(TokComma) {
			break
		}
	}
	if _, err := p.expect(TokRBrack); err != nil {
		return nil, fmt.Errorf("expected ']' after array literal")
	}
	return &Node{Kind: NodeArrayLit, ArrElems: elems}, nil
}

func (p *parser) parseObjectLit() (*Node, error) {
	p.advance() // consume "{"
	var props []ObjProp
	for !p.check(TokRBrace) && !p.atEnd() {
		if p.match(TokSpread) {
			expr, err := p.parseExpr()
			if err != nil {
				return nil, err
			}
			props = append(props, ObjProp{Value: expr, Spread: true})
		} else {
			var key *Node
			if p.check(TokIdent) {
				saved := p.pos
				name := p.advance()
				if p.check(TokColon) {
					key = &Node{Kind: NodeStringLit, StrVal: name.Str}
				} else {
					p.pos = saved
					key, _ = p.parseExpr()
				}
			} else {
				var err error
				key, err = p.parseExpr()
				if err != nil {
					return nil, err
				}
			}
			if _, err := p.expect(TokColon); err != nil {
				return nil, fmt.Errorf("expected ':' in object literal")
			}
			val, err := p.parseExpr()
			if err != nil {
				return nil, err
			}
			props = append(props, ObjProp{Key: key, Value: val})
		}
		if !p.match(TokComma) {
			break
		}
	}
	if _, err := p.expect(TokRBrace); err != nil {
		return nil, fmt.Errorf("expected '}' after object literal")
	}
	return &Node{Kind: NodeObjectLit, ObjProps: props}, nil
}
