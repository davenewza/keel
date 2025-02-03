package parser

import (
	"errors"
	"strings"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
	"github.com/teamkeel/keel/schema/node"
)

type Expression struct {
	node.Node
}

func (e *Expression) Parse(lex *lexer.PeekingLexer) error {
	parenCount := 0
	for {
		t := lex.Peek()

		if t.EOF() {
			e.EndPos = t.Pos
			return nil
		}

		if t.Value == ")" || t.Value == "]" {
			parenCount--
			if parenCount < 0 {
				e.EndPos = t.Pos
				return nil
			}
		}

		if t.Value == "(" || t.Value == "[" {
			parenCount++
		}

		if t.Value == "," && parenCount == 0 {
			e.EndPos = t.Pos
			return nil
		}

		t = lex.Next()
		e.Tokens = append(e.Tokens, *t)

		if len(e.Tokens) == 1 {
			e.Pos = t.Pos
		}
	}
}

func (e *Expression) String() string {
	if len(e.Tokens) == 0 {
		return ""
	}

	var result strings.Builder
	firstToken := e.Tokens[0]
	currentLine := e.Pos.Line
	currentColumn := e.Pos.Column

	// Handle first token
	if firstToken.Pos.Line > currentLine {
		// Add necessary newlines
		result.WriteString(strings.Repeat("\n", firstToken.Pos.Line-currentLine))
		// Reset column position for new line
		currentColumn = 0
	}
	// Add spaces to reach the correct column position
	if firstToken.Pos.Column > currentColumn {
		result.WriteString(strings.Repeat(" ", firstToken.Pos.Column-currentColumn))
	}
	result.WriteString(firstToken.Value)
	currentLine = firstToken.Pos.Line
	currentColumn = firstToken.Pos.Column + len(firstToken.Value)

	// Handle subsequent tokens
	for i := 1; i < len(e.Tokens); i++ {
		curr := e.Tokens[i]

		if curr.Pos.Line > currentLine {
			// Add necessary newlines
			result.WriteString(strings.Repeat("\n", curr.Pos.Line-currentLine))
			// Reset column position for new line
			currentColumn = 0
		}

		// Add spaces to reach the correct column position
		if curr.Pos.Column > currentColumn {
			result.WriteString(strings.Repeat(" ", curr.Pos.Column-currentColumn))
		}

		result.WriteString(curr.Value)
		currentLine = curr.Pos.Line
		currentColumn = curr.Pos.Column + len(curr.Value)
	}

	return result.String()
}

// CleanString removes new lines and unnecessary whitespaces, preserving single spaces between tokens
func (e *Expression) CleanString() string {
	v := ""
	for i, t := range e.Tokens {
		if i == 0 {
			v += t.Value
			continue
		}
		last := e.Tokens[i-1]
		hasWhitespace := (last.Pos.Offset + len(last.Value)) < t.Pos.Offset
		if hasWhitespace {
			v += " "
		}
		v += t.Value
	}
	return v
}

func ParseExpression(source string) (*Expression, error) {
	parser, err := participle.Build[Expression]()
	if err != nil {
		return nil, err
	}

	expr, err := parser.ParseString("", source)
	if err != nil {
		return nil, err
	}

	return expr, nil
}

type ExpressionIdent struct {
	node.Node

	Fragments []string
}

func (ident ExpressionIdent) String() string {
	return strings.Join(ident.Fragments, ".")
}

var ErrInvalidAssignmentExpression = errors.New("expression is not a valid assignment")

// ToAssignmentExpression splits an assignment expression into two separate expressions.
// E.g. the expression `post.age = 1 + 1` will become `post.age` and `1 + 1`
func (expr *Expression) ToAssignmentExpression() (*Expression, *Expression, error) {
	lhs := Expression{}
	lhs.Pos = expr.Pos
	lhs.Tokens = []lexer.Token{}
	assignmentAt := 0
	for i, token := range expr.Tokens {
		if token.Value == "=" {
			if i == 0 {
				return nil, nil, ErrInvalidAssignmentExpression
			}

			if i == len(expr.Tokens)-1 {
				return nil, nil, ErrInvalidAssignmentExpression
			}

			if expr.Tokens[i-1].Type > 0 || (expr.Tokens[i+1].Type > 0 && expr.Tokens[i+1].Type != 91) {
				return nil, nil, ErrInvalidAssignmentExpression
			}

			assignmentAt = i
			lhs.EndPos = token.Pos
			break
		}
		lhs.Tokens = append(lhs.Tokens, token)
	}

	if assignmentAt == 0 {
		return nil, nil, ErrInvalidAssignmentExpression
	}

	if len(expr.Tokens) == assignmentAt+1 {
		return nil, nil, ErrInvalidAssignmentExpression
	}

	rhs := Expression{}
	rhs.Pos = expr.Tokens[assignmentAt+1].Pos
	rhs.EndPos = expr.EndPos
	rhs.Tokens = []lexer.Token{}
	for i, token := range expr.Tokens {
		if i < assignmentAt+1 {
			continue
		}
		rhs.Tokens = append(rhs.Tokens, token)
	}

	return &lhs, &rhs, nil
}
