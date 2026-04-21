package intrinsicdsl

import (
	"fmt"
	"strconv"
)

type TokenKind int

const (
	TokNum TokenKind = iota
	TokStr
	TokIdent
	TokLParen
	TokRParen
	TokLBrack
	TokRBrack
	TokLBrace
	TokRBrace
	TokComma
	TokDot
	TokSemi
	TokColon
	TokArrow  // =>
	TokSpread // ...
	TokPlus
	TokMinus
	TokStar
	TokSlash
	TokPercent
	TokEq  // ==
	TokNEq // !=
	TokLt
	TokGt
	TokLEq    // <=
	TokGEq    // >=
	TokAnd    // &&
	TokOr     // ||
	TokNot    // !
	TokAssign // =
	TokQuestion
	TokEOF
)

var twoCharTokens = map[string]TokenKind{
	"=>": TokArrow, "==": TokEq, "!=": TokNEq, "<=": TokLEq,
	">=": TokGEq, "&&": TokAnd, "||": TokOr,
}

var singleCharTokens = map[byte]TokenKind{
	'(': TokLParen, ')': TokRParen, '[': TokLBrack, ']': TokRBrack,
	'{': TokLBrace, '}': TokRBrace, ',': TokComma, '.': TokDot,
	';': TokSemi, ':': TokColon, '+': TokPlus, '-': TokMinus,
	'*': TokStar, '/': TokSlash, '%': TokPercent, '<': TokLt,
	'>': TokGt, '!': TokNot, '=': TokAssign, '?': TokQuestion,
}

type Token struct {
	Kind TokenKind
	Num  float64
	Str  string
	Pos  int
}

type Lexer struct {
	src    string
	pos    int
	tokens []Token
	idx    int
}

func Lex(src string) ([]Token, error) {
	l := &Lexer{src: src}
	for {
		tok, err := l.next()
		if err != nil {
			return nil, err
		}
		l.tokens = append(l.tokens, tok)
		if tok.Kind == TokEOF {
			break
		}
	}
	return l.tokens, nil
}

func (l *Lexer) next() (Token, error) {
	l.skipWhitespace()
	if l.pos >= len(l.src) {
		return Token{Kind: TokEOF, Pos: l.pos}, nil
	}

	start := l.pos
	ch := l.src[l.pos]

	if ch >= '0' && ch <= '9' || ch == '.' && l.pos+1 < len(l.src) && l.src[l.pos+1] >= '0' && l.src[l.pos+1] <= '9' {
		return l.readNumber(start)
	}

	if ch == '\'' || ch == '"' || ch == '`' {
		return l.readString(start, ch)
	}

	if isIdentStart(ch) {
		return l.readIdent(start), nil
	}

	if l.pos+2 < len(l.src) && l.src[l.pos:l.pos+3] == "..." {
		l.pos += 3
		return Token{Kind: TokSpread, Pos: start}, nil
	}
	if l.pos+1 < len(l.src) {
		if kind, ok := twoCharTokens[l.src[l.pos:l.pos+2]]; ok {
			l.pos += 2
			return Token{Kind: kind, Pos: start}, nil
		}
	}

	l.pos++
	if kind, ok := singleCharTokens[ch]; ok {
		return Token{Kind: kind, Pos: start}, nil
	}

	return Token{}, fmt.Errorf("unexpected character %q at position %d", string(ch), start)
}

func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.src) {
		ch := l.src[l.pos]
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			l.pos++
		} else if ch == '/' && l.pos+1 < len(l.src) && l.src[l.pos+1] == '/' {
			// Line comment: skip to end of line
			l.pos += 2
			for l.pos < len(l.src) && l.src[l.pos] != '\n' {
				l.pos++
			}
		} else if ch == '/' && l.pos+1 < len(l.src) && l.src[l.pos+1] == '*' {
			// Block comment: skip to */
			l.pos += 2
			for l.pos+1 < len(l.src) {
				if l.src[l.pos] == '*' && l.src[l.pos+1] == '/' {
					l.pos += 2
					break
				}
				l.pos++
			}
		} else {
			break
		}
	}
}

func (l *Lexer) readNumber(start int) (Token, error) {
	for l.pos < len(l.src) && (l.src[l.pos] >= '0' && l.src[l.pos] <= '9' || l.src[l.pos] == '.') {
		l.pos++
	}
	n, err := strconv.ParseFloat(l.src[start:l.pos], 64)
	if err != nil {
		return Token{}, fmt.Errorf("invalid number at position %d", start)
	}
	return Token{Kind: TokNum, Num: n, Pos: start}, nil
}

func (l *Lexer) readString(start int, quote byte) (Token, error) {
	l.pos++
	var result []byte
	for l.pos < len(l.src) {
		ch := l.src[l.pos]
		if ch == quote {
			l.pos++
			return Token{Kind: TokStr, Str: string(result), Pos: start}, nil
		}
		if ch == '\\' && l.pos+1 < len(l.src) {
			l.pos++
			switch l.src[l.pos] {
			case 'n':
				result = append(result, '\n')
			case 't':
				result = append(result, '\t')
			case '\\':
				result = append(result, '\\')
			case '\'':
				result = append(result, '\'')
			case '"':
				result = append(result, '"')
			case '`':
				result = append(result, '`')
			default:
				result = append(result, l.src[l.pos])
			}
			l.pos++
			continue
		}
		result = append(result, ch)
		l.pos++
	}
	return Token{}, fmt.Errorf("unterminated string at position %d", start)
}

func (l *Lexer) readIdent(start int) Token {
	for l.pos < len(l.src) && isIdentCont(l.src[l.pos]) {
		l.pos++
	}
	text := l.src[start:l.pos]
	return Token{Kind: TokIdent, Str: text, Pos: start}
}

func isIdentStart(ch byte) bool {
	return ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch == '_' || ch == '$'
}

func isIdentCont(ch byte) bool {
	return isIdentStart(ch) || ch >= '0' && ch <= '9'
}
