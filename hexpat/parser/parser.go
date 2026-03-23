package parser

import (
	"strings"

	p "github.com/BlackBuck/pcom-go/parser"
	"github.com/BlackBuck/pcom-go/state"
)

// Parse parses a complete .hexpat file and returns the AST.
func Parse(input string) (*File, error) {
	st := state.NewState(input, state.Position{Offset: 0, Line: 1, Column: 1})
	// skip leading whitespace/comments
	skip().Run(&st)
	res, err := fileParser().Run(&st)
	if err.HasError() {
		return nil, &ParseError{
			Message:  err.Message,
			Expected: err.Expected,
			Got:      err.Got,
			Line:     err.Position.Line,
			Column:   err.Position.Column,
			Snippet:  err.Snippet,
		}
	}
	return &res.Value, nil
}

type ParseError struct {
	Message  string
	Expected string
	Got      string
	Line     int
	Column   int
	Snippet  string
}

func (e *ParseError) Error() string {
	return e.Message + " at line " + itoa(e.Line) + " col " + itoa(e.Column) +
		": expected " + e.Expected + ", got " + e.Got
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	s := ""
	neg := i < 0
	if neg {
		i = -i
	}
	for i > 0 {
		s = string(rune('0'+i%10)) + s
		i /= 10
	}
	if neg {
		s = "-" + s
	}
	return s
}

// --- File parser ---

func fileParser() p.Parser[File] {
	return p.Parser[File]{
		Run: func(curState *state.State) (p.Result[File], p.Error) {
			var items []Item
			for curState.InBounds(curState.Offset) {
				cp := curState.Save()
				skip().Run(curState)
				if !curState.InBounds(curState.Offset) {
					break
				}
				// Skip stray semicolons at top level
				if curState.Input[curState.Offset] == ';' {
					curState.Consume(1)
					continue
				}
				res, err := itemParser().Run(curState)
				if err.HasError() {
					return p.Result[File]{}, err
				}
				if res.Value != nil {
					items = append(items, res.Value)
				}
				// Make sure we made progress
				if curState.Offset == cp.Offset {
					return p.Result[File]{}, p.Error{
						Message:  "Parser stuck",
						Expected: "progress",
						Got:      string(curState.Input[curState.Offset]),
						Position: state.NewPositionFromState(curState),
						Snippet:  state.GetSnippetStringFromCurrentContext(curState),
					}
				}
			}
			return p.NewResult(File{Items: items}, curState,
				state.Span{Start: state.Position{Line: 1, Column: 1}, End: curState.Save()}), p.Error{}
		},
		Label: "file",
	}
}

// --- Top-level items ---

func itemParser() p.Parser[Item] {
	return p.Parser[Item]{
		Run: func(curState *state.State) (p.Result[Item], p.Error) {
			skip().Run(curState)
			if !curState.InBounds(curState.Offset) {
				return p.Result[Item]{}, p.Error{
					Message: "EOF", Expected: "item", Got: "EOF",
					Position: state.NewPositionFromState(curState),
				}
			}

			// Preprocessor directives
			if curState.Input[curState.Offset] == '#' {
				return mapResult(preprocessorParser(), curState)
			}

			// Peek at the next identifier to determine what to parse
			cp := curState.Save()
			idRes, idErr := identifier().Run(curState)
			curState.Rollback(cp)

			if idErr.HasError() {
				// Try variable declaration or expression statement
				return mapResult(varDeclOrExprStmt(), curState)
			}

			switch idRes.Value {
			case "import":
				return mapResult(importParser(), curState)
			case "struct":
				return mapResult(structParser(), curState)
			case "union":
				return mapResult(unionParser(), curState)
			case "enum":
				return mapResult(enumParser(), curState)
			case "bitfield":
				return mapResult(bitfieldParser(), curState)
			case "if":
				return mapItemFromStmt(ifStmtParser(), curState)
			case "while":
				return mapItemFromStmt(whileStmtParser(), curState)
			case "for":
				return mapItemFromStmt(forStmtParser(), curState)
			case "match":
				return mapItemFromStmt(matchStmtParser(), curState)
			case "using":
				return mapResult(usingParser(), curState)
			case "fn":
				return mapResult(fnParser(), curState)
			case "namespace":
				return mapResult(namespaceParser(), curState)
			case "const":
				// Skip "const" qualifier and parse as var decl
				keyword("const").Run(curState)
				skip().Run(curState)
				return mapResult(varDeclOrExprStmt(), curState)
			case "auto":
				// could be "auto namespace"
				cp2 := curState.Save()
				identifier().Run(curState)
				skip().Run(curState)
				nextRes, nextErr := identifier().Run(curState)
				curState.Rollback(cp2)
				if !nextErr.HasError() && nextRes.Value == "namespace" {
					return mapResult(namespaceParser(), curState)
				}
				return mapResult(varDeclOrExprStmt(), curState)
			default:
				// Variable declaration or expression statement
				return mapResult(varDeclOrExprStmt(), curState)
			}
		},
		Label: "item",
	}
}

// stmtItem wraps a Statement so it can be used as a top-level Item.
type stmtItem struct {
	Stmt Statement
}

func (stmtItem) itemNode() {}

func mapItemFromStmt[T Statement](parser p.Parser[T], curState *state.State) (p.Result[Item], p.Error) {
	res, err := parser.Run(curState)
	if err.HasError() {
		return p.Result[Item]{}, err
	}
	return p.Result[Item]{
		Value:     Item(stmtItem{Stmt: Statement(res.Value)}),
		NextState: res.NextState,
		Span:      res.Span,
	}, p.Error{}
}

func mapResult[T Item](parser p.Parser[T], curState *state.State) (p.Result[Item], p.Error) {
	res, err := parser.Run(curState)
	if err.HasError() {
		return p.Result[Item]{}, err
	}
	return p.Result[Item]{
		Value:     Item(res.Value),
		NextState: res.NextState,
		Span:      res.Span,
	}, p.Error{}
}

// --- Preprocessor ---

func preprocessorParser() p.Parser[Item] {
	return p.Parser[Item]{
		Run: func(curState *state.State) (p.Result[Item], p.Error) {
			if !curState.InBounds(curState.Offset) || curState.Input[curState.Offset] != '#' {
				return p.Result[Item]{}, p.Error{
					Message: "Expected #", Expected: "#",
					Position: state.NewPositionFromState(curState),
				}
			}
			cp := curState.Save()
			curState.Consume(1) // consume '#'

			res, err := identifier().Run(curState)
			if err.HasError() {
				curState.Rollback(cp)
				return p.Result[Item]{}, err
			}

			switch res.Value {
			case "pragma":
				return parsePragma(curState, cp)
			case "include":
				return parseInclude(curState, cp)
			case "define":
				return parseDefine(curState, cp)
			case "ifdef", "ifndef":
				return parseIfDef(curState, cp, res.Value == "ifndef")
			case "endif":
				// Return nil item — caller handles this as a terminator
				return p.NewResult[Item](nil, curState,
					state.Span{Start: cp, End: curState.Save()}), p.Error{}
			case "else":
				return p.NewResult[Item](nil, curState,
					state.Span{Start: cp, End: curState.Save()}), p.Error{}
			case "error":
				// consume rest of line
				end := curState.Offset
				for end < len(curState.Input) && curState.Input[end] != '\n' {
					end++
				}
				curState.Consume(end - curState.Offset)
				return p.NewResult[Item](nil, curState,
					state.Span{Start: cp, End: curState.Save()}), p.Error{}
			default:
				// Unknown directive - consume rest of line
				end := curState.Offset
				for end < len(curState.Input) && curState.Input[end] != '\n' {
					end++
				}
				curState.Consume(end - curState.Offset)
				return p.NewResult[Item](nil, curState,
					state.Span{Start: cp, End: curState.Save()}), p.Error{}
			}
		},
		Label: "preprocessor",
	}
}

func parsePragma(curState *state.State, start state.Position) (p.Result[Item], p.Error) {
	skip().Run(curState)
	// Read key — try identifier first, fall back to reading rest of line
	keyRes, err := identifier().Run(curState)
	var key, value string
	if err.HasError() {
		// Non-identifier pragma (e.g., #pragma 0.2 2023-10-29 ...)
		// Consume entire rest of line as key
		end := curState.Offset
		for end < len(curState.Input) && curState.Input[end] != '\n' {
			end++
		}
		key = strings.TrimSpace(curState.Input[curState.Offset:end])
		curState.Consume(end - curState.Offset)
	} else {
		key = keyRes.Value
		// Read rest of line as value
		skip().Run(curState)
		end := curState.Offset
		for end < len(curState.Input) && curState.Input[end] != '\n' {
			end++
		}
		value = strings.TrimSpace(curState.Input[curState.Offset:end])
		curState.Consume(end - curState.Offset)
	}
	return p.NewResult[Item](Pragma{Key: key, Value: value},
		curState, state.Span{Start: start, End: curState.Save()}), p.Error{}
}

func parseInclude(curState *state.State, start state.Position) (p.Result[Item], p.Error) {
	skip().Run(curState)
	if !curState.InBounds(curState.Offset) {
		return p.Result[Item]{}, p.Error{
			Message: "Expected include path", Expected: "path",
			Position: state.NewPositionFromState(curState),
		}
	}
	system := curState.Input[curState.Offset] == '<'
	var delim byte = '"'
	if system {
		delim = '>'
	}
	curState.Consume(1) // consume opening delimiter
	end := curState.Offset
	for end < len(curState.Input) && curState.Input[end] != delim {
		end++
	}
	path := curState.Input[curState.Offset:end]
	curState.Consume(end - curState.Offset + 1) // +1 for closing delimiter
	return p.NewResult[Item](Include{Path: path, System: system},
		curState, state.Span{Start: start, End: curState.Save()}), p.Error{}
}

func parseDefine(curState *state.State, start state.Position) (p.Result[Item], p.Error) {
	skip().Run(curState)
	nameRes, err := identifier().Run(curState)
	if err.HasError() {
		return p.Result[Item]{}, err
	}
	// Rest of line is value
	end := curState.Offset
	for end < len(curState.Input) && curState.Input[end] != '\n' {
		end++
	}
	value := strings.TrimSpace(curState.Input[curState.Offset:end])
	curState.Consume(end - curState.Offset)
	return p.NewResult[Item](Define{Name: nameRes.Value, Value: value},
		curState, state.Span{Start: start, End: curState.Save()}), p.Error{}
}

func parseIfDef(curState *state.State, start state.Position, negated bool) (p.Result[Item], p.Error) {
	skip().Run(curState)
	nameRes, err := identifier().Run(curState)
	if err.HasError() {
		return p.Result[Item]{}, err
	}

	// Parse body items until #endif or #else
	var body []Item
	var elseBody []Item
	inElse := false
	for {
		skip().Run(curState)
		if !curState.InBounds(curState.Offset) {
			break
		}
		// Check for #endif or #else
		if curState.Input[curState.Offset] == '#' {
			cp := curState.Save()
			curState.Consume(1)
			skipRes, _ := identifier().Run(curState)
			if skipRes.Value == "endif" {
				break
			}
			if skipRes.Value == "else" {
				inElse = true
				continue
			}
			curState.Rollback(cp)
		}
		itemRes, itemErr := itemParser().Run(curState)
		if itemErr.HasError() {
			return p.Result[Item]{}, itemErr
		}
		if itemRes.Value != nil {
			if inElse {
				elseBody = append(elseBody, itemRes.Value)
			} else {
				body = append(body, itemRes.Value)
			}
		}
	}

	return p.NewResult[Item](IfDef{Name: nameRes.Value, Negated: negated, Body: body, Else: elseBody},
		curState, state.Span{Start: start, End: curState.Save()}), p.Error{}
}

// --- Import ---

func importParser() p.Parser[Import] {
	return p.Parser[Import]{
		Run: func(curState *state.State) (p.Result[Import], p.Error) {
			cp := curState.Save()
			_, err := keyword("import").Run(curState)
			if err.HasError() {
				return p.Result[Import]{}, err
			}
			skip().Run(curState)

			// Check for: import * from "file" as Type
			if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == '*' {
				// import * from "file" as Alias;
				curState.Consume(1) // *
				skip().Run(curState)
				_, err = keyword("from").Run(curState)
				if err.HasError() {
					curState.Rollback(cp)
					return p.Result[Import]{}, err
				}
				skip().Run(curState)
				// Path can be a string literal or an identifier
				var importPath string
				strRes, strErr := stringLit().Run(curState)
				if !strErr.HasError() {
					importPath = strRes.Value.Value
				} else {
					idRes2, idErr2 := identifier().Run(curState)
					if idErr2.HasError() {
						curState.Rollback(cp)
						return p.Result[Import]{}, idErr2
					}
					importPath = idRes2.Value
				}
				skip().Run(curState)
				var alias string
				asRes, asErr := keyword("as").Run(curState)
				if !asErr.HasError() {
					_ = asRes
					skip().Run(curState)
					aliasRes, aliasErr := identifier().Run(curState)
					if !aliasErr.HasError() {
						alias = aliasRes.Value
					}
				}
				skip().Run(curState)
				consumeSemicolon(curState)
				return p.NewResult(Import{Path: []string{importPath}, Alias: alias},
					curState, state.Span{Start: cp, End: curState.Save()}), p.Error{}
			}

			// Normal: import std.mem;
			var path []string
			first, err := identifier().Run(curState)
			if err.HasError() {
				curState.Rollback(cp)
				return p.Result[Import]{}, err
			}
			path = append(path, first.Value)

			for {
				skip().Run(curState)
				if !curState.InBounds(curState.Offset) || curState.Input[curState.Offset] != '.' {
					break
				}
				curState.Consume(1) // '.'
				skip().Run(curState)
				// handle quoted segment for keyword conflicts
				if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == '"' {
					strRes, strErr := stringLit().Run(curState)
					if !strErr.HasError() {
						path = append(path, strRes.Value.Value)
						continue
					}
				}
				seg, segErr := identifier().Run(curState)
				if segErr.HasError() {
					break
				}
				path = append(path, seg.Value)
			}

			skip().Run(curState)
			var alias string
			_, asErr := keyword("as").Run(curState)
			if !asErr.HasError() {
				skip().Run(curState)
				aliasRes, aliasErr := identifier().Run(curState)
				if !aliasErr.HasError() {
					alias = aliasRes.Value
				}
			}

			skip().Run(curState)
			consumeSemicolon(curState)

			return p.NewResult(Import{Path: path, Alias: alias},
				curState, state.Span{Start: cp, End: curState.Save()}), p.Error{}
		},
		Label: "import",
	}
}

// --- Type references ---

func typeParser() p.Parser[TypeNode] {
	return p.Parser[TypeNode]{
		Run: func(curState *state.State) (p.Result[TypeNode], p.Error) {
			cp := curState.Save()

			// Check for endian prefix
			idRes, idErr := identifier().Run(curState)
			if idErr.HasError() {
				return p.Result[TypeNode]{}, idErr
			}

			if idRes.Value == "le" || idRes.Value == "be" {
				skip().Run(curState)
				inner, innerErr := typeParser().Run(curState)
				if innerErr.HasError() {
					curState.Rollback(cp)
					return p.Result[TypeNode]{}, innerErr
				}
				return p.NewResult[TypeNode](EndianType{Order: idRes.Value, Inner: inner.Value},
					curState, state.Span{Start: cp, End: curState.Save()}), p.Error{}
			}

			// Check if builtin
			if isBuiltinType(idRes.Value) {
				return p.NewResult[TypeNode](BuiltinType{Name: idRes.Value},
					curState, state.Span{Start: cp, End: curState.Save()}), p.Error{}
			}

			// Named type, possibly with namespace and template args
			var ns []string
			name := idRes.Value

			for {
				skip().Run(curState)
				// Check for :: namespace separator
				if curState.InBounds(curState.Offset+1) &&
					curState.Input[curState.Offset] == ':' &&
					curState.Input[curState.Offset+1] == ':' {
					curState.Consume(2)
					skip().Run(curState)
					ns = append(ns, name)
					nextRes, nextErr := identifier().Run(curState)
					if nextErr.HasError() {
						curState.Rollback(cp)
						return p.Result[TypeNode]{}, nextErr
					}
					name = nextRes.Value
				} else {
					break
				}
			}

			// Check for template args <T, U>
			var typeArgs []TypeNode
			skip().Run(curState)
			if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == '<' {
				curState.Consume(1)
				for {
					skip().Run(curState)
					if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == '>' {
						curState.Consume(1)
						break
					}
					// Try parsing a type arg — could be a type or an auto expression
					argRes, argErr := typeParser().Run(curState)
					if argErr.HasError() {
						// Might be a non-type arg (number, string, expression) —
						// capture the raw text as a RawTypeArg
						rawStart := curState.Offset
						for curState.InBounds(curState.Offset) &&
							curState.Input[curState.Offset] != ',' &&
							curState.Input[curState.Offset] != '>' {
							curState.Consume(1)
						}
						rawText := string(curState.Input[rawStart:curState.Offset])
						if rawText != "" {
							typeArgs = append(typeArgs, RawTypeArg{Text: rawText})
						}
					} else {
						typeArgs = append(typeArgs, argRes.Value)
					}
					skip().Run(curState)
					if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == ',' {
						curState.Consume(1)
					}
				}
			}

			return p.NewResult[TypeNode](NamedType{Namespace: ns, Name: name, TypeArgs: typeArgs},
				curState, state.Span{Start: cp, End: curState.Save()}), p.Error{}
		},
		Label: "type",
	}
}

var builtinTypes = map[string]bool{
	"u8": true, "u16": true, "u24": true, "u32": true, "u48": true, "u64": true,
	"u96": true, "u128": true,
	"s8": true, "s16": true, "s24": true, "s32": true, "s48": true, "s64": true,
	"s96": true, "s128": true,
	"float": true, "double": true, "f32": true, "f64": true,
	"char": true, "char16": true, "bool": true, "str": true, "auto": true,
}

func isBuiltinType(name string) bool {
	return builtinTypes[name]
}

// --- Expressions ---

func exprParser() p.Parser[Expr] {
	return p.Lazy("expression", func() p.Parser[Expr] {
		return ternaryExpr()
	})
}

func ternaryExpr() p.Parser[Expr] {
	return p.Parser[Expr]{
		Run: func(curState *state.State) (p.Result[Expr], p.Error) {
			cp := curState.Save()
			left, err := logicalOrExpr().Run(curState)
			if err.HasError() {
				return p.Result[Expr]{}, err
			}
			skip().Run(curState)
			if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == '?' {
				curState.Consume(1)
				skip().Run(curState)
				then, thenErr := exprParser().Run(curState)
				if thenErr.HasError() {
					curState.Rollback(cp)
					return p.Result[Expr]{}, thenErr
				}
				skip().Run(curState)
				if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == ':' {
					curState.Consume(1)
					skip().Run(curState)
					els, elsErr := exprParser().Run(curState)
					if elsErr.HasError() {
						curState.Rollback(cp)
						return p.Result[Expr]{}, elsErr
					}
					return p.NewResult[Expr](TernaryExpr{Cond: left.Value, Then: then.Value, Else: els.Value},
						curState, state.Span{Start: cp, End: curState.Save()}), p.Error{}
				}
			}
			return left, p.Error{}
		},
		Label: "ternary",
	}
}

func logicalOrExpr() p.Parser[Expr] {
	return binaryExprParser("logical or", logicalAndExpr(), []string{"||"})
}

func logicalAndExpr() p.Parser[Expr] {
	return binaryExprParser("logical and", bitwiseOrExpr(), []string{"&&"})
}

func bitwiseOrExpr() p.Parser[Expr] {
	return binaryExprParser("bitwise or", bitwiseXorExpr(), []string{"|"})
}

func bitwiseXorExpr() p.Parser[Expr] {
	return binaryExprParser("bitwise xor", bitwiseAndExpr(), []string{"^"})
}

func bitwiseAndExpr() p.Parser[Expr] {
	return binaryExprParser("bitwise and", equalityExpr(), []string{"&"})
}

func equalityExpr() p.Parser[Expr] {
	return binaryExprParser("equality", comparisonExpr(), []string{"==", "!="})
}

func comparisonExpr() p.Parser[Expr] {
	return binaryExprParser("comparison", shiftExpr(), []string{"<=", ">=", "<", ">"})
}

func shiftExpr() p.Parser[Expr] {
	return binaryExprParser("shift", addExpr(), []string{"<<", ">>"})
}

func addExpr() p.Parser[Expr] {
	return binaryExprParser("add", mulExpr(), []string{"+", "-"})
}

func mulExpr() p.Parser[Expr] {
	return binaryExprParser("mul", unaryExpr(), []string{"*", "/", "%"})
}

func binaryExprParser(label string, operand p.Parser[Expr], ops []string) p.Parser[Expr] {
	return p.Parser[Expr]{
		Run: func(curState *state.State) (p.Result[Expr], p.Error) {
			left, err := operand.Run(curState)
			if err.HasError() {
				return p.Result[Expr]{}, err
			}
			for {
				skip().Run(curState)
				matched := false
				for _, op := range ops {
					if matchOp(curState, op) {
						cp := curState.Save()
						curState.Consume(len(op))
						skip().Run(curState)
						right, rightErr := operand.Run(curState)
						if rightErr.HasError() {
							curState.Rollback(cp)
							return left, p.Error{}
						}
						left = p.Result[Expr]{
							Value:     BinaryExpr{Op: op, Left: left.Value, Right: right.Value},
							NextState: curState,
							Span:      state.Span{Start: left.Span.Start, End: curState.Save()},
						}
						matched = true
						break
					}
				}
				if !matched {
					break
				}
			}
			return left, p.Error{}
		},
		Label: label,
	}
}

// matchOp checks if the operator matches at current position, avoiding matching
// prefixes (e.g. "<" should not match "<<" or "<=").
func matchOp(curState *state.State, op string) bool {
	if !curState.InBounds(curState.Offset + len(op) - 1) {
		return false
	}
	if curState.Input[curState.Offset:curState.Offset+len(op)] != op {
		return false
	}
	// Avoid matching prefix: if the next char extends the operator, don't match
	nextIdx := curState.Offset + len(op)
	if nextIdx < len(curState.Input) {
		next := curState.Input[nextIdx]
		switch op {
		case "|":
			if next == '|' {
				return false
			}
		case "&":
			if next == '&' {
				return false
			}
		case "<":
			if next == '<' || next == '=' {
				return false
			}
		case ">":
			if next == '>' || next == '=' {
				return false
			}
		case "!":
			if next == '=' {
				return false
			}
		case "=":
			if next == '=' {
				return false
			}
		}
	}
	return true
}

func unaryExpr() p.Parser[Expr] {
	return p.Parser[Expr]{
		Run: func(curState *state.State) (p.Result[Expr], p.Error) {
			if !curState.InBounds(curState.Offset) {
				return p.Result[Expr]{}, p.Error{
					Message: "Expected expression", Expected: "expression", Got: "EOF",
					Position: state.NewPositionFromState(curState),
				}
			}
			c := curState.Input[curState.Offset]
			cp := curState.Save()

			// Unary operators
			switch c {
			case '!':
				if curState.InBounds(curState.Offset+1) && curState.Input[curState.Offset+1] == '=' {
					// != operator, not unary
					break
				}
				curState.Consume(1)
				skip().Run(curState)
				operand, err := unaryExpr().Run(curState)
				if err.HasError() {
					curState.Rollback(cp)
					return p.Result[Expr]{}, err
				}
				return p.NewResult[Expr](UnaryExpr{Op: "!", Operand: operand.Value, Prefix: true},
					curState, state.Span{Start: cp, End: curState.Save()}), p.Error{}
			case '~':
				curState.Consume(1)
				skip().Run(curState)
				operand, err := unaryExpr().Run(curState)
				if err.HasError() {
					curState.Rollback(cp)
					return p.Result[Expr]{}, err
				}
				return p.NewResult[Expr](UnaryExpr{Op: "~", Operand: operand.Value, Prefix: true},
					curState, state.Span{Start: cp, End: curState.Save()}), p.Error{}
			case '-':
				// Could be negative number literal or unary minus
				// Try negative number first
				numRes, numErr := floatLit().Run(curState)
				if !numErr.HasError() {
					return postfixExpr(p.NewResult[Expr](numRes.Value, curState, numRes.Span), curState)
				}
				curState.Rollback(cp)
				intRes, intErr := numberLit().Run(curState)
				if !intErr.HasError() {
					return postfixExpr(p.NewResult[Expr](intRes.Value, curState, intRes.Span), curState)
				}
				curState.Rollback(cp)
				// Unary minus on expression
				curState.Consume(1)
				skip().Run(curState)
				operand, err := unaryExpr().Run(curState)
				if err.HasError() {
					curState.Rollback(cp)
					return p.Result[Expr]{}, err
				}
				return p.NewResult[Expr](UnaryExpr{Op: "-", Operand: operand.Value, Prefix: true},
					curState, state.Span{Start: cp, End: curState.Save()}), p.Error{}
			}

			return primaryExpr().Run(curState)
		},
		Label: "unary",
	}
}

func primaryExpr() p.Parser[Expr] {
	return p.Parser[Expr]{
		Run: func(curState *state.State) (p.Result[Expr], p.Error) {
			if !curState.InBounds(curState.Offset) {
				return p.Result[Expr]{}, p.Error{
					Message: "Expected expression", Expected: "expression", Got: "EOF",
					Position: state.NewPositionFromState(curState),
				}
			}
			cp := curState.Save()
			c := curState.Input[curState.Offset]

			// Array initializer: { expr, expr, ... }
			if c == '{' {
				curState.Consume(1)
				var elems []Expr
				for {
					skip().Run(curState)
					if !curState.InBounds(curState.Offset) || curState.Input[curState.Offset] == '}' {
						break
					}
					elemRes, elemErr := exprParser().Run(curState)
					if elemErr.HasError() {
						break
					}
					elems = append(elems, elemRes.Value)
					skip().Run(curState)
					if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == ',' {
						curState.Consume(1)
					}
				}
				if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == '}' {
					curState.Consume(1)
				}
				return p.NewResult[Expr](ArrayInitExpr{Elements: elems}, curState,
					state.Span{Start: cp, End: curState.Save()}), p.Error{}
			}

			// Dollar
			if c == '$' {
				curState.Consume(1)
				res := p.NewResult[Expr](DollarExpr{}, curState, state.Span{Start: cp, End: curState.Save()})
				return postfixExpr(res, curState)
			}

			// Parenthesized expression
			if c == '(' {
				curState.Consume(1)
				skip().Run(curState)
				inner, err := exprParser().Run(curState)
				if err.HasError() {
					curState.Rollback(cp)
					return p.Result[Expr]{}, err
				}
				skip().Run(curState)
				if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == ')' {
					curState.Consume(1)
				}
				return postfixExpr(p.NewResult[Expr](inner.Value, curState,
					state.Span{Start: cp, End: curState.Save()}), curState)
			}

			// String literal
			if c == '"' {
				res, err := stringLit().Run(curState)
				if !err.HasError() {
					return postfixExpr(p.NewResult[Expr](res.Value, curState, res.Span), curState)
				}
			}

			// Char literal
			if c == '\'' {
				res, err := charLit().Run(curState)
				if !err.HasError() {
					return postfixExpr(p.NewResult[Expr](res.Value, curState, res.Span), curState)
				}
			}

			// Float literal (try before int since "1.5" starts with digit)
			if isDigit(c) {
				floatRes, floatErr := floatLit().Run(curState)
				if !floatErr.HasError() {
					return postfixExpr(p.NewResult[Expr](floatRes.Value, curState, floatRes.Span), curState)
				}
				curState.Rollback(cp)
			}

			// Number literal
			if isDigit(c) {
				res, err := numberLit().Run(curState)
				if !err.HasError() {
					return postfixExpr(p.NewResult[Expr](res.Value, curState, res.Span), curState)
				}
				curState.Rollback(cp)
			}

			// Keywords and identifiers
			if isIdentStart(c) {
				idRes, idErr := identifier().Run(curState)
				if idErr.HasError() {
					return p.Result[Expr]{}, idErr
				}

				switch idRes.Value {
				case "true":
					return postfixExpr(p.NewResult[Expr](BoolLit{Value: true}, curState,
						state.Span{Start: cp, End: curState.Save()}), curState)
				case "false":
					return postfixExpr(p.NewResult[Expr](BoolLit{Value: false}, curState,
						state.Span{Start: cp, End: curState.Save()}), curState)
				case "null":
					return postfixExpr(p.NewResult[Expr](NumberLit{Value: 0, Raw: "null"}, curState,
						state.Span{Start: cp, End: curState.Save()}), curState)
				case "this":
					return postfixExpr(p.NewResult[Expr](Ident{Name: "this"}, curState,
						state.Span{Start: cp, End: curState.Save()}), curState)
				case "parent":
					return postfixExpr(p.NewResult[Expr](Ident{Name: "parent"}, curState,
						state.Span{Start: cp, End: curState.Save()}), curState)
				case "sizeof":
					skip().Run(curState)
					if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == '(' {
						curState.Consume(1)
						skip().Run(curState)
						inner, innerErr := exprParser().Run(curState)
						if innerErr.HasError() {
							curState.Rollback(cp)
							return p.Result[Expr]{}, innerErr
						}
						skip().Run(curState)
						if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == ')' {
							curState.Consume(1)
						}
						return postfixExpr(p.NewResult[Expr](SizeOfExpr{Operand: inner.Value}, curState,
							state.Span{Start: cp, End: curState.Save()}), curState)
					}
				case "addressof":
					skip().Run(curState)
					if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == '(' {
						curState.Consume(1)
						skip().Run(curState)
						inner, innerErr := exprParser().Run(curState)
						if innerErr.HasError() {
							curState.Rollback(cp)
							return p.Result[Expr]{}, innerErr
						}
						skip().Run(curState)
						if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == ')' {
							curState.Consume(1)
						}
						return postfixExpr(p.NewResult[Expr](AddressOfExpr{Operand: inner.Value}, curState,
							state.Span{Start: cp, End: curState.Save()}), curState)
					}
				case "while":
					// while(cond) used in array sizes
					skip().Run(curState)
					if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == '(' {
						curState.Consume(1)
						skip().Run(curState)
						cond, condErr := exprParser().Run(curState)
						if condErr.HasError() {
							curState.Rollback(cp)
							return p.Result[Expr]{}, condErr
						}
						skip().Run(curState)
						if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == ')' {
							curState.Consume(1)
						}
						return p.NewResult[Expr](WhileExpr{Cond: cond.Value}, curState,
							state.Span{Start: cp, End: curState.Save()}), p.Error{}
					}
				case "be", "le":
					// Endian-qualified cast: be u32(15) or le u16(0)
					skip().Run(curState)
					castTypeRes, castTypeErr := typeParser().Run(curState)
					if !castTypeErr.HasError() {
						endianType := EndianType{Order: idRes.Value, Inner: castTypeRes.Value}
						skip().Run(curState)
						if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == '(' {
							curState.Consume(1)
							skip().Run(curState)
							inner, innerErr := exprParser().Run(curState)
							if !innerErr.HasError() {
								skip().Run(curState)
								if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == ')' {
									curState.Consume(1)
								}
								return postfixExpr(p.NewResult[Expr](CastExpr{Type: endianType, Operand: inner.Value}, curState,
									state.Span{Start: cp, End: curState.Save()}), curState)
							}
						}
						// Just an endian type reference without call
						return postfixExpr(p.NewResult[Expr](Ident{Name: idRes.Value}, curState,
							state.Span{Start: cp, End: curState.Save()}), curState)
					}
					curState.Rollback(cp)
					// Fall through to normal identifier handling
				}

				// Check for :: namespace access
				var expr Expr = Ident{Name: idRes.Value}
				for {
					skip().Run(curState)
					if curState.InBounds(curState.Offset+1) &&
						curState.Input[curState.Offset] == ':' &&
						curState.Input[curState.Offset+1] == ':' {
						curState.Consume(2)
						skip().Run(curState)
						nextRes, nextErr := identifier().Run(curState)
						if nextErr.HasError() {
							break
						}
						expr = NamespaceAccess{Namespace: exprName(expr), Member: Ident{Name: nextRes.Value}}
					} else {
						break
					}
				}

				// Check for function call: ident(args)
				skip().Run(curState)
				if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == '(' {
					curState.Consume(1)
					skip().Run(curState)
					var args []Expr
					if !curState.InBounds(curState.Offset) || curState.Input[curState.Offset] != ')' {
						for {
							skip().Run(curState)
							arg, argErr := exprParser().Run(curState)
							if argErr.HasError() {
								break
							}
							args = append(args, arg.Value)
							skip().Run(curState)
							if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == ',' {
								curState.Consume(1)
							} else {
								break
							}
						}
					}
					skip().Run(curState)
					if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == ')' {
						curState.Consume(1)
					}
					expr = FnCall{Func: expr, Args: args}
				}

				return postfixExpr(p.NewResult[Expr](expr, curState,
					state.Span{Start: cp, End: curState.Save()}), curState)
			}

			return p.Result[Expr]{}, p.Error{
				Message:  "Expected expression",
				Expected: "expression",
				Got:      string(c),
				Position: state.NewPositionFromState(curState),
				Snippet:  state.GetSnippetStringFromCurrentContext(curState),
			}
		},
		Label: "primary expression",
	}
}

// postfixExpr handles member access (.field), indexing ([expr]), and cast operations.
func postfixExpr(base p.Result[Expr], curState *state.State) (p.Result[Expr], p.Error) {
	for {
		skip().Run(curState)
		if !curState.InBounds(curState.Offset) {
			break
		}
		c := curState.Input[curState.Offset]

		if c == '.' && !(curState.InBounds(curState.Offset+1) && curState.Input[curState.Offset+1] == '.') {
			curState.Consume(1)
			skip().Run(curState)
			memberRes, memberErr := identifier().Run(curState)
			if memberErr.HasError() {
				break
			}
			// Check if this member is itself a function call
			skip().Run(curState)
			if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == '(' {
				curState.Consume(1)
				skip().Run(curState)
				var args []Expr
				if !curState.InBounds(curState.Offset) || curState.Input[curState.Offset] != ')' {
					for {
						arg, argErr := exprParser().Run(curState)
						if argErr.HasError() {
							break
						}
						args = append(args, arg.Value)
						skip().Run(curState)
						if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == ',' {
							curState.Consume(1)
							skip().Run(curState)
						} else {
							break
						}
					}
				}
				if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == ')' {
					curState.Consume(1)
				}
				base = p.Result[Expr]{
					Value:     FnCall{Func: MemberAccess{Object: base.Value, Member: memberRes.Value}, Args: args},
					NextState: curState,
					Span:      state.Span{Start: base.Span.Start, End: curState.Save()},
				}
			} else {
				base = p.Result[Expr]{
					Value:     MemberAccess{Object: base.Value, Member: memberRes.Value},
					NextState: curState,
					Span:      state.Span{Start: base.Span.Start, End: curState.Save()},
				}
			}
			continue
		}

		if c == '[' && !(curState.InBounds(curState.Offset+1) && curState.Input[curState.Offset+1] == '[') {
			curState.Consume(1)
			skip().Run(curState)
			idx, idxErr := exprParser().Run(curState)
			if idxErr.HasError() {
				break
			}
			skip().Run(curState)
			if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == ']' {
				curState.Consume(1)
			}
			base = p.Result[Expr]{
				Value:     IndexAccess{Object: base.Value, Index: idx.Value},
				NextState: curState,
				Span:      state.Span{Start: base.Span.Start, End: curState.Save()},
			}
			continue
		}

		break
	}
	return base, p.Error{}
}

func exprName(e Expr) string {
	switch v := e.(type) {
	case Ident:
		return v.Name
	case NamespaceAccess:
		return v.Namespace + "::" + exprName(v.Member)
	default:
		return ""
	}
}

// --- Struct definition ---

func structParser() p.Parser[StructDef] {
	return p.Parser[StructDef]{
		Run: func(curState *state.State) (p.Result[StructDef], p.Error) {
			cp := curState.Save()
			_, err := keyword("struct").Run(curState)
			if err.HasError() {
				return p.Result[StructDef]{}, err
			}
			skip().Run(curState)

			nameRes, nameErr := identifier().Run(curState)
			if nameErr.HasError() {
				curState.Rollback(cp)
				return p.Result[StructDef]{}, nameErr
			}
			skip().Run(curState)

			// Template params
			var typeParams []string
			if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == '<' {
				typeParams = parseTemplateParams(curState)
			}
			skip().Run(curState)

			// Inheritance
			var parent string
			if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == ':' {
				curState.Consume(1)
				skip().Run(curState)
				parentRes, parentErr := identifier().Run(curState)
				if !parentErr.HasError() {
					parent = parentRes.Value
				}
			}
			skip().Run(curState)

			// Body
			if !curState.InBounds(curState.Offset) || curState.Input[curState.Offset] != '{' {
				curState.Rollback(cp)
				return p.Result[StructDef]{}, p.Error{
					Message: "Expected {", Expected: "{",
					Position: state.NewPositionFromState(curState),
				}
			}
			curState.Consume(1)

			body := parseBody(curState)

			skip().Run(curState)
			if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == '}' {
				curState.Consume(1)
			}

			// Handle #ifdef blocks between } and ; (e.g. #ifdef __IMHEX__ [[attr]] #endif)
			skip().Run(curState)
			skipIfDefBlocks(curState)

			// Parse attributes after closing brace
			skip().Run(curState)
			attrs := parseAttributes(curState)

			// Consume optional semicolon
			skip().Run(curState)
			consumeSemicolon(curState)

			return p.NewResult(StructDef{
				Name: nameRes.Value, TypeParams: typeParams,
				Parent: parent, Body: body, Attrs: attrs,
			}, curState, state.Span{Start: cp, End: curState.Save()}), p.Error{}
		},
		Label: "struct",
	}
}

// --- Union definition ---

func unionParser() p.Parser[UnionDef] {
	return p.Parser[UnionDef]{
		Run: func(curState *state.State) (p.Result[UnionDef], p.Error) {
			cp := curState.Save()
			_, err := keyword("union").Run(curState)
			if err.HasError() {
				return p.Result[UnionDef]{}, err
			}
			skip().Run(curState)

			nameRes, nameErr := identifier().Run(curState)
			if nameErr.HasError() {
				curState.Rollback(cp)
				return p.Result[UnionDef]{}, nameErr
			}
			skip().Run(curState)

			var typeParams []string
			if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == '<' {
				typeParams = parseTemplateParams(curState)
			}
			skip().Run(curState)

			if !curState.InBounds(curState.Offset) || curState.Input[curState.Offset] != '{' {
				curState.Rollback(cp)
				return p.Result[UnionDef]{}, p.Error{
					Message: "Expected {", Expected: "{",
					Position: state.NewPositionFromState(curState),
				}
			}
			curState.Consume(1)

			body := parseBody(curState)

			skip().Run(curState)
			if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == '}' {
				curState.Consume(1)
			}

			skip().Run(curState)
			attrs := parseAttributes(curState)
			skip().Run(curState)
			consumeSemicolon(curState)

			return p.NewResult(UnionDef{
				Name: nameRes.Value, TypeParams: typeParams,
				Body: body, Attrs: attrs,
			}, curState, state.Span{Start: cp, End: curState.Save()}), p.Error{}
		},
		Label: "union",
	}
}

// --- Enum definition ---

func enumParser() p.Parser[EnumDef] {
	return p.Parser[EnumDef]{
		Run: func(curState *state.State) (p.Result[EnumDef], p.Error) {
			cp := curState.Save()
			_, err := keyword("enum").Run(curState)
			if err.HasError() {
				return p.Result[EnumDef]{}, err
			}
			skip().Run(curState)

			// Name is optional (anonymous enums)
			var enumName string
			nameRes, nameErr := identifier().Run(curState)
			if !nameErr.HasError() {
				enumName = nameRes.Value
			}
			skip().Run(curState)

			// : underlying_type
			var underlyingType TypeNode
			if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == ':' {
				curState.Consume(1)
				skip().Run(curState)
				typeRes, typeErr := typeParser().Run(curState)
				if !typeErr.HasError() {
					underlyingType = typeRes.Value
				}
			}
			skip().Run(curState)

			if !curState.InBounds(curState.Offset) || curState.Input[curState.Offset] != '{' {
				curState.Rollback(cp)
				return p.Result[EnumDef]{}, p.Error{
					Message: "Expected {", Expected: "{",
					Position: state.NewPositionFromState(curState),
				}
			}
			curState.Consume(1)

			// Parse members
			var members []EnumMember
			for {
				skip().Run(curState)
				if !curState.InBounds(curState.Offset) || curState.Input[curState.Offset] == '}' {
					break
				}
				memberRes, memberErr := identifier().Run(curState)
				if memberErr.HasError() {
					break
				}
				member := EnumMember{Name: memberRes.Value}
				skip().Run(curState)
				if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == '=' {
					curState.Consume(1)
					skip().Run(curState)
					valRes, valErr := exprParser().Run(curState)
					if !valErr.HasError() {
						member.Value = valRes.Value
					}
					// Check for range: ...
					skip().Run(curState)
					if curState.InBounds(curState.Offset+2) &&
						curState.Input[curState.Offset:curState.Offset+3] == "..." {
						curState.Consume(3)
						skip().Run(curState)
						endRes, endErr := exprParser().Run(curState)
						if !endErr.HasError() {
							member.RangeEnd = endRes.Value
						}
					}
				}
				members = append(members, member)
				skip().Run(curState)
				if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == ',' {
					curState.Consume(1)
				}
			}

			skip().Run(curState)
			if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == '}' {
				curState.Consume(1)
			}

			skip().Run(curState)
			attrs := parseAttributes(curState)
			skip().Run(curState)
			consumeSemicolon(curState)

			return p.NewResult(EnumDef{
				Name: enumName, UnderlyingType: underlyingType,
				Members: members, Attrs: attrs,
			}, curState, state.Span{Start: cp, End: curState.Save()}), p.Error{}
		},
		Label: "enum",
	}
}

// --- Bitfield definition ---

func bitfieldParser() p.Parser[BitfieldDef] {
	return p.Parser[BitfieldDef]{
		Run: func(curState *state.State) (p.Result[BitfieldDef], p.Error) {
			cp := curState.Save()
			_, err := keyword("bitfield").Run(curState)
			if err.HasError() {
				return p.Result[BitfieldDef]{}, err
			}
			skip().Run(curState)

			nameRes, nameErr := identifier().Run(curState)
			if nameErr.HasError() {
				curState.Rollback(cp)
				return p.Result[BitfieldDef]{}, nameErr
			}
			skip().Run(curState)

			// Template params
			var typeParams []string
			if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == '<' {
				typeParams = parseTemplateParams(curState)
			}
			skip().Run(curState)

			if !curState.InBounds(curState.Offset) || curState.Input[curState.Offset] != '{' {
				curState.Rollback(cp)
				return p.Result[BitfieldDef]{}, p.Error{
					Message: "Expected {", Expected: "{",
					Position: state.NewPositionFromState(curState),
				}
			}
			curState.Consume(1)

			entries := parseBitfieldBody(curState)

			skip().Run(curState)
			if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == '}' {
				curState.Consume(1)
			}

			skip().Run(curState)
			attrs := parseAttributes(curState)
			skip().Run(curState)
			consumeSemicolon(curState)

			return p.NewResult(BitfieldDef{
				Name: nameRes.Value, TypeParams: typeParams, Body: entries, Attrs: attrs,
			}, curState, state.Span{Start: cp, End: curState.Save()}), p.Error{}
		},
		Label: "bitfield",
	}
}

// parseBitfieldMatch parses a match block inside a bitfield, where arm bodies
// contain bitfield entries (name : bits) rather than regular statements.
func parseBitfieldMatch(curState *state.State) (MatchStmt, p.Error) {
	cp := curState.Save()
	_, err := keyword("match").Run(curState)
	if err.HasError() {
		return MatchStmt{}, err
	}
	skip().Run(curState)

	// Match arguments
	var args []Expr
	if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == '(' {
		curState.Consume(1)
		for {
			skip().Run(curState)
			if !curState.InBounds(curState.Offset) || curState.Input[curState.Offset] == ')' {
				break
			}
			argRes, argErr := exprParser().Run(curState)
			if argErr.HasError() {
				break
			}
			args = append(args, argRes.Value)
			skip().Run(curState)
			if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == ',' {
				curState.Consume(1)
			}
		}
		if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == ')' {
			curState.Consume(1)
		}
	}
	skip().Run(curState)

	if !curState.InBounds(curState.Offset) || curState.Input[curState.Offset] != '{' {
		curState.Rollback(cp)
		return MatchStmt{}, p.Error{Message: "Expected {", Expected: "{", Position: state.NewPositionFromState(curState)}
	}
	curState.Consume(1)

	// Parse arms with bitfield-style bodies
	var arms []MatchArm
	for {
		skip().Run(curState)
		if !curState.InBounds(curState.Offset) || curState.Input[curState.Offset] == '}' {
			break
		}
		if curState.Input[curState.Offset] != '(' {
			break
		}
		curState.Consume(1)

		var patterns []MatchPattern
		for {
			skip().Run(curState)
			if !curState.InBounds(curState.Offset) || curState.Input[curState.Offset] == ')' {
				break
			}
			if curState.Input[curState.Offset] == '_' {
				curState.Consume(1)
				patterns = append(patterns, MatchPattern{Wildcard: true})
			} else {
				patRes, patErr := exprParser().Run(curState)
				if patErr.HasError() {
					break
				}
				pat := MatchPattern{Value: patRes.Value}
				skip().Run(curState)
				if curState.InBounds(curState.Offset+2) &&
					curState.Input[curState.Offset:curState.Offset+3] == "..." {
					curState.Consume(3)
					skip().Run(curState)
					endRes, endErr := exprParser().Run(curState)
					if !endErr.HasError() {
						pat.RangeEnd = endRes.Value
					}
				}
				patterns = append(patterns, pat)
			}
			skip().Run(curState)
			if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == ',' {
				curState.Consume(1)
			}
		}
		if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == ')' {
			curState.Consume(1)
		}
		skip().Run(curState)
		if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == ':' {
			curState.Consume(1)
		}
		skip().Run(curState)

		// Arm body: bitfield entries (type name : bits;) or a block
		// Parse as bitfield entries until next ( or }
		arm := MatchArm{Patterns: patterns}
		if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == '{' {
			curState.Consume(1)
			_ = parseBitfieldBody(curState)
			skip().Run(curState)
			if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == '}' {
				curState.Consume(1)
			}
		} else {
			// Single bitfield entry: [type] name : bits [attrs] ;
			// Just consume until ; and move on
			for curState.InBounds(curState.Offset) &&
				curState.Input[curState.Offset] != ';' &&
				curState.Input[curState.Offset] != '}' {
				curState.Consume(1)
			}
			consumeSemicolon(curState)
		}
		arms = append(arms, arm)
	}

	skip().Run(curState)
	if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == '}' {
		curState.Consume(1)
	}

	return MatchStmt{Args: args, Arms: arms}, p.Error{}
}

func parseBitfieldBody(curState *state.State) []BitfieldEntry {
	var entries []BitfieldEntry
	for {
		skip().Run(curState)
		if !curState.InBounds(curState.Offset) || curState.Input[curState.Offset] == '}' {
			break
		}

		// Skip stray semicolons
		if curState.Input[curState.Offset] == ';' {
			curState.Consume(1)
			continue
		}

		entryCP := curState.Save()

		// Peek for keyword-based constructs
		if isIdentStart(curState.Input[curState.Offset]) {
			kwCP := curState.Save()
			kwRes, kwErr := identifier().Run(curState)
			curState.Rollback(kwCP)

			if !kwErr.HasError() {
				switch kwRes.Value {
				case "if":
					// Conditional inside bitfield — skip over it using parseBlock
					ifRes, ifErr := ifStmtParser().Run(curState)
					if !ifErr.HasError() {
						_ = ifRes
						// We can't represent if-stmt as a BitfieldEntry cleanly,
						// but we need to consume it. Add a placeholder.
						entries = append(entries, BitfieldEntry{Name: "__if__"})
						continue
					}
					curState.Rollback(entryCP)
				case "match":
					matchRes, matchErr := parseBitfieldMatch(curState)
					if !matchErr.HasError() {
						_ = matchRes
						entries = append(entries, BitfieldEntry{Name: "__match__"})
						continue
					}
					curState.Rollback(entryCP)
				case "padding":
					// padding : N; in bitfield
					keyword("padding").Run(curState)
					skip().Run(curState)
					if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == ':' {
						curState.Consume(1)
						skip().Run(curState)
						bitsRes, bitsErr := exprParser().Run(curState)
						if !bitsErr.HasError() {
							skip().Run(curState)
							attrs := parseAttributes(curState)
							skip().Run(curState)
							consumeSemicolon(curState)
							entries = append(entries, BitfieldEntry{Name: "padding", Bits: bitsRes.Value, Attrs: attrs})
							continue
						}
					} else if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == '[' {
						// padding[N] form inside bitfield
						curState.Rollback(entryCP)
						padRes, padErr := paddingParser().Run(curState)
						if !padErr.HasError() {
							entries = append(entries, BitfieldEntry{Name: "padding", Bits: padRes.Value.Size})
							continue
						}
					}
					curState.Rollback(entryCP)
				}
			}
		}
		curState.Rollback(entryCP)

		// Check for 'signed' or 'bool' keyword before field name
		var entryType TypeNode
		signedCP := curState.Save()
		kwRes, kwErr := identifier().Run(curState)
		if !kwErr.HasError() && (kwRes.Value == "signed" || kwRes.Value == "bool" || kwRes.Value == "unsigned") {
			if kwRes.Value == "bool" {
				entryType = BuiltinType{Name: "bool"}
			}
			skip().Run(curState)
		} else {
			curState.Rollback(signedCP)
		}

		// Try to parse as: type name (= expr | : bits) ; (typed var/field decl in bitfield)
		typedCP := curState.Save()
		typeRes, typeErr := typeParser().Run(curState)
		if !typeErr.HasError() {
			skip().Run(curState)
			nameRes, nameErr := identifier().Run(curState)
			if !nameErr.HasError() {
				skip().Run(curState)
				if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == '=' &&
					(!curState.InBounds(curState.Offset+1) || curState.Input[curState.Offset+1] != '=') {
					// type name = expr;
					curState.Consume(1)
					skip().Run(curState)
					valRes, valErr := exprParser().Run(curState)
					if !valErr.HasError() {
						skip().Run(curState)
						attrs := parseAttributes(curState)
						skip().Run(curState)
						consumeSemicolon(curState)
						entries = append(entries, BitfieldEntry{Name: nameRes.Value, Type: typeRes.Value, Attrs: attrs, Bits: valRes.Value})
						continue
					}
				} else if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == ':' &&
					(!curState.InBounds(curState.Offset+1) || curState.Input[curState.Offset+1] != ':') {
					// type name : bits;
					curState.Consume(1)
					skip().Run(curState)
					bitsRes, bitsErr := exprParser().Run(curState)
					if !bitsErr.HasError() {
						skip().Run(curState)
						attrs := parseAttributes(curState)
						skip().Run(curState)
						consumeSemicolon(curState)
						entries = append(entries, BitfieldEntry{Name: nameRes.Value, Type: typeRes.Value, Attrs: attrs, Bits: bitsRes.Value})
						continue
					}
				}
			}
		}
		curState.Rollback(typedCP)

		// Try expression statement (e.g. std::assert(...); inside bitfield)
		{
			exprCP := curState.Save()
			exprRes, exprErr := exprParser().Run(curState)
			if !exprErr.HasError() {
				skip().Run(curState)
				if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == ';' {
					consumeSemicolon(curState)
					_ = exprRes
					entries = append(entries, BitfieldEntry{Name: "__expr__"})
					continue
				}
			}
			curState.Rollback(exprCP)
		}

		// Standard bitfield entry: name : bits; or name = expr;
		idRes, idErr := identifier().Run(curState)
		if idErr.HasError() {
			// Can't parse — skip to next ; or }
			for curState.InBounds(curState.Offset) &&
				curState.Input[curState.Offset] != ';' &&
				curState.Input[curState.Offset] != '}' {
				curState.Consume(1)
			}
			consumeSemicolon(curState)
			continue
		}
		entry := BitfieldEntry{Name: idRes.Value, Type: entryType}

		skip().Run(curState)
		if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == ':' &&
			(!curState.InBounds(curState.Offset+1) || curState.Input[curState.Offset+1] != ':') {
			curState.Consume(1)
			skip().Run(curState)
			bitsRes, bitsErr := exprParser().Run(curState)
			if !bitsErr.HasError() {
				entry.Bits = bitsRes.Value
			}
		} else if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == '=' &&
			(!curState.InBounds(curState.Offset+1) || curState.Input[curState.Offset+1] != '=') {
			// Computed field: name = expr;
			curState.Consume(1)
			skip().Run(curState)
			valRes, valErr := exprParser().Run(curState)
			if !valErr.HasError() {
				entry.Bits = valRes.Value
			}
		}
		skip().Run(curState)
		entry.Attrs = parseAttributes(curState)
		entries = append(entries, entry)
		skip().Run(curState)
		consumeSemicolon(curState)
	}
	return entries
}

// --- Using definition ---

func usingParser() p.Parser[UsingDef] {
	return p.Parser[UsingDef]{
		Run: func(curState *state.State) (p.Result[UsingDef], p.Error) {
			cp := curState.Save()
			_, err := keyword("using").Run(curState)
			if err.HasError() {
				return p.Result[UsingDef]{}, err
			}
			skip().Run(curState)

			nameRes, nameErr := identifier().Run(curState)
			if nameErr.HasError() {
				curState.Rollback(cp)
				return p.Result[UsingDef]{}, nameErr
			}
			skip().Run(curState)

			// Template params
			var typeParams []string
			if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == '<' {
				typeParams = parseTemplateParams(curState)
			}
			skip().Run(curState)

			// Forward declaration: using Foo; (no = sign)
			if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == ';' {
				curState.Consume(1)
				return p.NewResult(UsingDef{Name: nameRes.Value, TypeParams: typeParams},
					curState, state.Span{Start: cp, End: curState.Save()}), p.Error{}
			}

			// using Name = Type;
			if !curState.InBounds(curState.Offset) || curState.Input[curState.Offset] != '=' {
				curState.Rollback(cp)
				return p.Result[UsingDef]{}, p.Error{
					Message: "Expected = in using", Expected: "=",
					Position: state.NewPositionFromState(curState),
				}
			}
			curState.Consume(1)
			skip().Run(curState)

			typeRes, typeErr := typeParser().Run(curState)
			if typeErr.HasError() {
				curState.Rollback(cp)
				return p.Result[UsingDef]{}, typeErr
			}

			skip().Run(curState)
			attrs := parseAttributes(curState)
			skip().Run(curState)
			consumeSemicolon(curState)

			return p.NewResult(UsingDef{
				Name: nameRes.Value, TypeParams: typeParams,
				Type: typeRes.Value, Attrs: attrs,
			}, curState, state.Span{Start: cp, End: curState.Save()}), p.Error{}
		},
		Label: "using",
	}
}

// --- Function definition ---

func fnParser() p.Parser[FnDef] {
	return p.Parser[FnDef]{
		Run: func(curState *state.State) (p.Result[FnDef], p.Error) {
			cp := curState.Save()
			_, err := keyword("fn").Run(curState)
			if err.HasError() {
				return p.Result[FnDef]{}, err
			}
			skip().Run(curState)

			nameRes, nameErr := identifier().Run(curState)
			if nameErr.HasError() {
				curState.Rollback(cp)
				return p.Result[FnDef]{}, nameErr
			}
			skip().Run(curState)

			// Parameters
			var params []FnParam
			if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == '(' {
				curState.Consume(1)
				for {
					skip().Run(curState)
					if !curState.InBounds(curState.Offset) || curState.Input[curState.Offset] == ')' {
						break
					}
					param := FnParam{}
					// Check for 'ref' keyword
					paramCP := curState.Save()
					refRes, refErr := keyword("ref").Run(curState)
					if !refErr.HasError() {
						param.Ref = true
						_ = refRes
						skip().Run(curState)
					} else {
						curState.Rollback(paramCP)
					}

					typeRes, typeErr := typeParser().Run(curState)
					if typeErr.HasError() {
						break
					}
					param.Type = typeRes.Value
					skip().Run(curState)
					paramNameRes, paramNameErr := identifier().Run(curState)
					if !paramNameErr.HasError() {
						param.Name = paramNameRes.Value
					}
					skip().Run(curState)
					// Default value: = expr
					if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == '=' &&
						(!curState.InBounds(curState.Offset+1) || curState.Input[curState.Offset+1] != '=') {
						curState.Consume(1)
						skip().Run(curState)
						defRes, defErr := exprParser().Run(curState)
						if !defErr.HasError() {
							param.Default = defRes.Value
						}
					}
					params = append(params, param)
					skip().Run(curState)
					if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == ',' {
						curState.Consume(1)
					}
				}
				if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == ')' {
					curState.Consume(1)
				}
			}
			skip().Run(curState)

			// Body
			if !curState.InBounds(curState.Offset) || curState.Input[curState.Offset] != '{' {
				curState.Rollback(cp)
				return p.Result[FnDef]{}, p.Error{
					Message: "Expected { in fn", Expected: "{",
					Position: state.NewPositionFromState(curState),
				}
			}
			curState.Consume(1)

			body := parseBody(curState)

			skip().Run(curState)
			if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == '}' {
				curState.Consume(1)
			}
			skip().Run(curState)
			consumeSemicolon(curState)

			return p.NewResult(FnDef{
				Name: nameRes.Value, Params: params, Body: body,
			}, curState, state.Span{Start: cp, End: curState.Save()}), p.Error{}
		},
		Label: "fn",
	}
}

// --- Namespace ---

func namespaceParser() p.Parser[NamespaceDef] {
	return p.Parser[NamespaceDef]{
		Run: func(curState *state.State) (p.Result[NamespaceDef], p.Error) {
			cp := curState.Save()

			// Check for "auto namespace"
			auto := false
			autoRes, autoErr := keyword("auto").Run(curState)
			if !autoErr.HasError() {
				auto = true
				_ = autoRes
				skip().Run(curState)
			} else {
				curState.Rollback(cp)
			}

			_, err := keyword("namespace").Run(curState)
			if err.HasError() {
				curState.Rollback(cp)
				return p.Result[NamespaceDef]{}, err
			}
			skip().Run(curState)

			nameRes, nameErr := identifier().Run(curState)
			if nameErr.HasError() {
				curState.Rollback(cp)
				return p.Result[NamespaceDef]{}, nameErr
			}
			skip().Run(curState)

			if !curState.InBounds(curState.Offset) || curState.Input[curState.Offset] != '{' {
				curState.Rollback(cp)
				return p.Result[NamespaceDef]{}, p.Error{
					Message: "Expected {", Expected: "{",
					Position: state.NewPositionFromState(curState),
				}
			}
			curState.Consume(1)

			// Parse namespace body as items
			var items []Item
			for {
				skip().Run(curState)
				if !curState.InBounds(curState.Offset) || curState.Input[curState.Offset] == '}' {
					break
				}
				itemRes, itemErr := itemParser().Run(curState)
				if itemErr.HasError() {
					break
				}
				if itemRes.Value != nil {
					items = append(items, itemRes.Value)
				}
			}

			skip().Run(curState)
			if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == '}' {
				curState.Consume(1)
			}

			return p.NewResult(NamespaceDef{
				Name: nameRes.Value, Auto: auto, Items: items,
			}, curState, state.Span{Start: cp, End: curState.Save()}), p.Error{}
		},
		Label: "namespace",
	}
}

// --- Statement parsing (for struct/union/function bodies) ---

func parseBody(curState *state.State) []Statement {
	var stmts []Statement
	for {
		skip().Run(curState)
		if !curState.InBounds(curState.Offset) || curState.Input[curState.Offset] == '}' {
			break
		}
		cp := curState.Save()
		stmt, err := statementParser().Run(curState)
		if err.HasError() {
			// Try to recover by skipping to next semicolon or closing brace
			for curState.InBounds(curState.Offset) &&
				curState.Input[curState.Offset] != ';' &&
				curState.Input[curState.Offset] != '}' {
				curState.Consume(1)
			}
			if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == ';' {
				curState.Consume(1)
			}
			continue
		}
		if curState.Offset == cp.Offset {
			// No progress, skip a character
			curState.Consume(1)
			continue
		}
		stmts = append(stmts, stmt.Value)
	}
	return stmts
}

func statementParser() p.Parser[Statement] {
	return p.Parser[Statement]{
		Run: func(curState *state.State) (p.Result[Statement], p.Error) {
			skip().Run(curState)
			if !curState.InBounds(curState.Offset) {
				return p.Result[Statement]{}, p.Error{
					Message: "EOF", Expected: "statement", Got: "EOF",
					Position: state.NewPositionFromState(curState),
				}
			}

			c := curState.Input[curState.Offset]
			cp := curState.Save()

			// Try preprocessor directive in body
			if c == '#' {
				ppRes, ppErr := preprocessorParser().Run(curState)
				if !ppErr.HasError() {
					// Wrap in ExprStmt with nil — preprocessor in body is a no-op for now
					_ = ppRes
					return p.NewResult[Statement](ExprStmt{}, curState,
						state.Span{Start: cp, End: curState.Save()}), p.Error{}
				}
				curState.Rollback(cp)
			}

			// Peek at identifier for keyword-based statements
			if isIdentStart(c) {
				idRes, idErr := identifier().Run(curState)
				curState.Rollback(cp)
				if !idErr.HasError() {
					switch idRes.Value {
					case "if":
						return mapStmt(ifStmtParser(), curState)
					case "while":
						return mapStmt(whileStmtParser(), curState)
					case "for":
						return mapStmt(forStmtParser(), curState)
					case "match":
						return mapStmt(matchStmtParser(), curState)
					case "return":
						return mapStmt(returnStmtParser(), curState)
					case "break":
						keyword("break").Run(curState)
						skip().Run(curState)
						consumeSemicolon(curState)
						return p.NewResult[Statement](BreakStmt{}, curState,
							state.Span{Start: cp, End: curState.Save()}), p.Error{}
					case "continue":
						keyword("continue").Run(curState)
						skip().Run(curState)
						consumeSemicolon(curState)
						return p.NewResult[Statement](ContinueStmt{}, curState,
							state.Span{Start: cp, End: curState.Save()}), p.Error{}
					case "padding":
						return mapStmt(paddingParser(), curState)
					case "try":
						return mapStmt(tryCatchParser(), curState)
					case "struct":
						// Inline struct definition — treat as an item within body
						res, err := structParser().Run(curState)
						if !err.HasError() {
							return p.NewResult[Statement](ExprStmt{}, curState, res.Span), p.Error{}
						}
						curState.Rollback(cp)
					case "union":
						res, err := unionParser().Run(curState)
						if !err.HasError() {
							return p.NewResult[Statement](ExprStmt{}, curState, res.Span), p.Error{}
						}
						curState.Rollback(cp)
					case "enum":
						res, err := enumParser().Run(curState)
						if !err.HasError() {
							return p.NewResult[Statement](ExprStmt{}, curState, res.Span), p.Error{}
						}
						curState.Rollback(cp)
					case "fn":
						res, err := fnParser().Run(curState)
						if !err.HasError() {
							return p.NewResult[Statement](ExprStmt{}, curState, res.Span), p.Error{}
						}
						curState.Rollback(cp)
					case "using":
						res, err := usingParser().Run(curState)
						if !err.HasError() {
							return p.NewResult[Statement](ExprStmt{}, curState, res.Span), p.Error{}
						}
						curState.Rollback(cp)
					case "const":
						// Skip "const" qualifier and parse as var decl
						keyword("const").Run(curState)
						skip().Run(curState)
						cp3 := curState.Save()
						res, err := varDeclParser().Run(curState)
						if !err.HasError() {
							return p.NewResult[Statement](res.Value, curState, res.Span), p.Error{}
						}
						curState.Rollback(cp3)
					case "namespace":
						res, err := namespaceParser().Run(curState)
						if !err.HasError() {
							return p.NewResult[Statement](ExprStmt{}, curState, res.Span), p.Error{}
						}
						curState.Rollback(cp)
					case "bitfield":
						res, err := bitfieldParser().Run(curState)
						if !err.HasError() {
							return p.NewResult[Statement](ExprStmt{}, curState, res.Span), p.Error{}
						}
						curState.Rollback(cp)
					}
				}
			}

			// Try variable declaration: Type name [...] [@ expr] [[attrs]] ;
			{
				cp2 := curState.Save()
				res, err := varDeclParser().Run(curState)
				if !err.HasError() {
					return p.NewResult[Statement](res.Value, curState, res.Span), p.Error{}
				}
				curState.Rollback(cp2)
			}

			// Dollar assignment: $ = expr; or $ += expr;
			if c == '$' {
				res, err := dollarAssignParser().Run(curState)
				if !err.HasError() {
					return p.NewResult[Statement](res.Value, curState, res.Span), p.Error{}
				}
				curState.Rollback(cp)
			}

			// Expression statement (function calls, assignments)
			{
				exprRes, exprErr := exprParser().Run(curState)
				if !exprErr.HasError() {
					skip().Run(curState)
					// Check for assignment
					if curState.InBounds(curState.Offset) {
						for _, op := range []string{"=", "+=", "-=", "*=", "/=", "%=", "<<=", ">>=", "&=", "|=", "^="} {
							if len(op) <= len(curState.Input)-curState.Offset &&
								curState.Input[curState.Offset:curState.Offset+len(op)] == op &&
								(op != "=" || !curState.InBounds(curState.Offset+1) || curState.Input[curState.Offset+1] != '=') {
								curState.Consume(len(op))
								skip().Run(curState)
								valRes, valErr := exprParser().Run(curState)
								if !valErr.HasError() {
									skip().Run(curState)
									consumeSemicolon(curState)
									return p.NewResult[Statement](
										AssignStmt{Target: exprRes.Value, Op: op, Value: valRes.Value},
										curState, state.Span{Start: cp, End: curState.Save()}), p.Error{}
								}
							}
						}
					}
					skip().Run(curState)
					consumeSemicolon(curState)
					return p.NewResult[Statement](ExprStmt{Expr: exprRes.Value}, curState,
						state.Span{Start: cp, End: curState.Save()}), p.Error{}
				}
			}

			return p.Result[Statement]{}, p.Error{
				Message:  "Expected statement",
				Expected: "statement",
				Got:      string(c),
				Position: state.NewPositionFromState(curState),
				Snippet:  state.GetSnippetStringFromCurrentContext(curState),
			}
		},
		Label: "statement",
	}
}

func mapStmt[T Statement](parser p.Parser[T], curState *state.State) (p.Result[Statement], p.Error) {
	res, err := parser.Run(curState)
	if err.HasError() {
		return p.Result[Statement]{}, err
	}
	return p.Result[Statement]{
		Value: Statement(res.Value), NextState: res.NextState, Span: res.Span,
	}, p.Error{}
}

// --- Variable declaration ---

func varDeclParser() p.Parser[VarDecl] {
	return p.Parser[VarDecl]{
		Run: func(curState *state.State) (p.Result[VarDecl], p.Error) {
			cp := curState.Save()

			// Parse type
			typeRes, typeErr := typeParser().Run(curState)
			if typeErr.HasError() {
				return p.Result[VarDecl]{}, typeErr
			}
			skip().Run(curState)

			// Check for pointer: *name
			var ptrInfo *PointerInfo
			if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == '*' {
				curState.Consume(1)
				skip().Run(curState)
				ptrInfo = &PointerInfo{}
			}

			// Parse name
			nameRes, nameErr := identifier().Run(curState)
			if nameErr.HasError() {
				curState.Rollback(cp)
				return p.Result[VarDecl]{}, nameErr
			}
			skip().Run(curState)

			// Array size: [expr] — but not [[ which is attributes
			var arraySize Expr
			if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == '[' &&
				!(curState.InBounds(curState.Offset+1) && curState.Input[curState.Offset+1] == '[') {
				curState.Consume(1)
				skip().Run(curState)
				if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == ']' {
					// Unsized array: Type name[]
					curState.Consume(1)
				} else {
					sizeRes, sizeErr := exprParser().Run(curState)
					if !sizeErr.HasError() {
						arraySize = sizeRes.Value
					}
					skip().Run(curState)
					if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == ']' {
						curState.Consume(1)
					}
				}
			}
			skip().Run(curState)

			// Pointer size type: : u32
			if ptrInfo != nil {
				if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == ':' {
					curState.Consume(1)
					skip().Run(curState)
					sizeTypeRes, sizeTypeErr := typeParser().Run(curState)
					if !sizeTypeErr.HasError() {
						ptrInfo.SizeType = sizeTypeRes.Value
					}
				}
				skip().Run(curState)
			}

			// Initializer: = expr
			var initExpr Expr
			if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == '=' &&
				(!curState.InBounds(curState.Offset+1) || curState.Input[curState.Offset+1] != '=') {
				curState.Consume(1)
				skip().Run(curState)
				initRes, initErr := exprParser().Run(curState)
				if !initErr.HasError() {
					initExpr = initRes.Value
				}
			}
			skip().Run(curState)

			// Offset: @ expr
			var offset Expr
			if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == '@' {
				curState.Consume(1)
				skip().Run(curState)
				offRes, offErr := exprParser().Run(curState)
				if !offErr.HasError() {
					offset = offRes.Value
				}
			}
			skip().Run(curState)

			// Attributes
			attrs := parseAttributes(curState)
			skip().Run(curState)

			consumeSemicolon(curState)

			return p.NewResult(VarDecl{
				Type: typeRes.Value, Name: nameRes.Value,
				Array: arraySize, Pointer: ptrInfo,
				Offset: offset, Init: initExpr, Attrs: attrs,
			}, curState, state.Span{Start: cp, End: curState.Save()}), p.Error{}
		},
		Label: "var decl",
	}
}

func varDeclOrExprStmt() p.Parser[Item] {
	return p.Parser[Item]{
		Run: func(curState *state.State) (p.Result[Item], p.Error) {
			cp := curState.Save()
			// Try var decl first
			res, err := varDeclParser().Run(curState)
			if !err.HasError() {
				return p.Result[Item]{
					Value: Item(res.Value), NextState: curState, Span: res.Span,
				}, p.Error{}
			}
			curState.Rollback(cp)

			// Try expression statement
			exprRes, exprErr := exprParser().Run(curState)
			if !exprErr.HasError() {
				skip().Run(curState)
				// Check for assignment
				if curState.InBounds(curState.Offset) {
					for _, op := range []string{"=", "+=", "-=", "*=", "/="} {
						if len(op) <= len(curState.Input)-curState.Offset &&
							curState.Input[curState.Offset:curState.Offset+len(op)] == op &&
							(op != "=" || !curState.InBounds(curState.Offset+1) || curState.Input[curState.Offset+1] != '=') {
							curState.Consume(len(op))
							skip().Run(curState)
							valRes, valErr := exprParser().Run(curState)
							if !valErr.HasError() {
								skip().Run(curState)
								consumeSemicolon(curState)
								stmt := AssignStmt{Target: exprRes.Value, Op: op, Value: valRes.Value}
								return p.Result[Item]{
									Value:     Item(VarDecl{}), // placeholder
									NextState: curState,
								}, p.Error{}
								_ = stmt
							}
						}
					}
				}
				consumeSemicolon(curState)
				return p.Result[Item]{
					Value:     Item(VarDecl{Name: "__expr__"}), // placeholder
					NextState: curState,
				}, p.Error{}
			}
			curState.Rollback(cp)

			return p.Result[Item]{}, p.Error{
				Message: "Expected declaration or statement",
				Expected: "declaration",
				Position: state.NewPositionFromState(curState),
				Snippet:  state.GetSnippetStringFromCurrentContext(curState),
			}
		},
		Label: "var decl or expr",
	}
}

// --- Control flow ---

func ifStmtParser() p.Parser[IfStmt] {
	return p.Parser[IfStmt]{
		Run: func(curState *state.State) (p.Result[IfStmt], p.Error) {
			cp := curState.Save()
			_, err := keyword("if").Run(curState)
			if err.HasError() {
				return p.Result[IfStmt]{}, err
			}
			skip().Run(curState)

			// Condition (may or may not be parenthesized)
			cond, condErr := parenExpr(curState)
			if condErr.HasError() {
				curState.Rollback(cp)
				return p.Result[IfStmt]{}, condErr
			}
			skip().Run(curState)

			// Then block
			thenBody := parseBlock(curState)
			skip().Run(curState)

			// Else
			var elseBody []Statement
			elseCP := curState.Save()
			_, elseErr := keyword("else").Run(curState)
			if !elseErr.HasError() {
				skip().Run(curState)
				// Check for else-if
				ifCP := curState.Save()
				_, ifErr := keyword("if").Run(curState)
				curState.Rollback(ifCP)
				if !ifErr.HasError() {
					// else if — parse as nested if
					nestedRes, nestedErr := ifStmtParser().Run(curState)
					if !nestedErr.HasError() {
						elseBody = []Statement{nestedRes.Value}
					}
				} else {
					elseBody = parseBlock(curState)
				}
			} else {
				curState.Rollback(elseCP)
			}

			return p.NewResult(IfStmt{Cond: cond, Then: thenBody, Else: elseBody},
				curState, state.Span{Start: cp, End: curState.Save()}), p.Error{}
		},
		Label: "if",
	}
}

func whileStmtParser() p.Parser[WhileStmt] {
	return p.Parser[WhileStmt]{
		Run: func(curState *state.State) (p.Result[WhileStmt], p.Error) {
			cp := curState.Save()
			_, err := keyword("while").Run(curState)
			if err.HasError() {
				return p.Result[WhileStmt]{}, err
			}
			skip().Run(curState)

			cond, condErr := parenExpr(curState)
			if condErr.HasError() {
				curState.Rollback(cp)
				return p.Result[WhileStmt]{}, condErr
			}
			skip().Run(curState)

			body := parseBlock(curState)

			return p.NewResult(WhileStmt{Cond: cond, Body: body},
				curState, state.Span{Start: cp, End: curState.Save()}), p.Error{}
		},
		Label: "while",
	}
}

func forStmtParser() p.Parser[ForStmt] {
	return p.Parser[ForStmt]{
		Run: func(curState *state.State) (p.Result[ForStmt], p.Error) {
			cp := curState.Save()
			_, err := keyword("for").Run(curState)
			if err.HasError() {
				return p.Result[ForStmt]{}, err
			}
			skip().Run(curState)

			// (init; cond; post)
			if !curState.InBounds(curState.Offset) || curState.Input[curState.Offset] != '(' {
				curState.Rollback(cp)
				return p.Result[ForStmt]{}, p.Error{
					Message: "Expected (", Expected: "(",
					Position: state.NewPositionFromState(curState),
				}
			}
			curState.Consume(1)
			skip().Run(curState)

			// ImHex for loops use commas as separators: for(init, cond, post)
			// But also support semicolons: for(init; cond; post)
			initRes, _ := statementParser().Run(curState)
			skip().Run(curState)
			consumeCommaOrSemicolon(curState)
			skip().Run(curState)

			condRes, _ := exprParser().Run(curState)
			skip().Run(curState)
			consumeCommaOrSemicolon(curState)
			skip().Run(curState)

			postRes, _ := statementParser().Run(curState)
			skip().Run(curState)

			if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == ')' {
				curState.Consume(1)
			}
			skip().Run(curState)

			body := parseBlock(curState)

			return p.NewResult(ForStmt{
				Init: initRes.Value, Cond: condRes.Value,
				Post: postRes.Value, Body: body,
			}, curState, state.Span{Start: cp, End: curState.Save()}), p.Error{}
		},
		Label: "for",
	}
}

func matchStmtParser() p.Parser[MatchStmt] {
	return p.Parser[MatchStmt]{
		Run: func(curState *state.State) (p.Result[MatchStmt], p.Error) {
			cp := curState.Save()
			_, err := keyword("match").Run(curState)
			if err.HasError() {
				return p.Result[MatchStmt]{}, err
			}
			skip().Run(curState)

			// Match arguments (value1, value2, ...)
			var args []Expr
			if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == '(' {
				curState.Consume(1)
				for {
					skip().Run(curState)
					if !curState.InBounds(curState.Offset) || curState.Input[curState.Offset] == ')' {
						break
					}
					argRes, argErr := exprParser().Run(curState)
					if argErr.HasError() {
						break
					}
					args = append(args, argRes.Value)
					skip().Run(curState)
					if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == ',' {
						curState.Consume(1)
					}
				}
				if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == ')' {
					curState.Consume(1)
				}
			}
			skip().Run(curState)

			// Match body { arms }
			if !curState.InBounds(curState.Offset) || curState.Input[curState.Offset] != '{' {
				curState.Rollback(cp)
				return p.Result[MatchStmt]{}, p.Error{
					Message: "Expected {", Expected: "{",
					Position: state.NewPositionFromState(curState),
				}
			}
			curState.Consume(1)

			var arms []MatchArm
			for {
				skip().Run(curState)
				if !curState.InBounds(curState.Offset) || curState.Input[curState.Offset] == '}' {
					break
				}

				// Parse pattern: (val1, val2, ...) or (_, val)
				if !curState.InBounds(curState.Offset) || curState.Input[curState.Offset] != '(' {
					break
				}
				curState.Consume(1)

				var patterns []MatchPattern
				for {
					skip().Run(curState)
					if !curState.InBounds(curState.Offset) || curState.Input[curState.Offset] == ')' {
						break
					}
					if curState.Input[curState.Offset] == '_' {
						curState.Consume(1)
						patterns = append(patterns, MatchPattern{Wildcard: true})
					} else {
						patRes, patErr := exprParser().Run(curState)
						if patErr.HasError() {
							break
						}
						pat := MatchPattern{Value: patRes.Value}
						// Check for range pattern: value ... end
						skip().Run(curState)
						if curState.InBounds(curState.Offset+2) &&
							curState.Input[curState.Offset:curState.Offset+3] == "..." {
							curState.Consume(3)
							skip().Run(curState)
							endRes, endErr := exprParser().Run(curState)
							if !endErr.HasError() {
								pat.RangeEnd = endRes.Value
							}
						}
						patterns = append(patterns, pat)
					}
					skip().Run(curState)
					if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == ',' {
						curState.Consume(1)
					}
				}
				if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == ')' {
					curState.Consume(1)
				}
				skip().Run(curState)

				// : is optional separator
				if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == ':' {
					curState.Consume(1)
				}
				skip().Run(curState)

				// Arm body — either a block or single statement
				armBody := parseMatchArmBody(curState)
				arms = append(arms, MatchArm{Patterns: patterns, Body: armBody})
			}

			skip().Run(curState)
			if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == '}' {
				curState.Consume(1)
			}

			return p.NewResult(MatchStmt{Args: args, Arms: arms},
				curState, state.Span{Start: cp, End: curState.Save()}), p.Error{}
		},
		Label: "match",
	}
}

func parseMatchArmBody(curState *state.State) []Statement {
	// Could be a braced block or a single statement (var decl typically)
	if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == '{' {
		return parseBlock(curState)
	}
	// Single statement
	res, err := statementParser().Run(curState)
	if err.HasError() {
		return nil
	}
	return []Statement{res.Value}
}

func returnStmtParser() p.Parser[ReturnStmt] {
	return p.Parser[ReturnStmt]{
		Run: func(curState *state.State) (p.Result[ReturnStmt], p.Error) {
			cp := curState.Save()
			_, err := keyword("return").Run(curState)
			if err.HasError() {
				return p.Result[ReturnStmt]{}, err
			}
			skip().Run(curState)

			var val Expr
			if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] != ';' {
				valRes, valErr := exprParser().Run(curState)
				if !valErr.HasError() {
					val = valRes.Value
				}
			}
			skip().Run(curState)
			consumeSemicolon(curState)

			return p.NewResult(ReturnStmt{Value: val}, curState,
				state.Span{Start: cp, End: curState.Save()}), p.Error{}
		},
		Label: "return",
	}
}

func paddingParser() p.Parser[PaddingStmt] {
	return p.Parser[PaddingStmt]{
		Run: func(curState *state.State) (p.Result[PaddingStmt], p.Error) {
			cp := curState.Save()
			_, err := keyword("padding").Run(curState)
			if err.HasError() {
				return p.Result[PaddingStmt]{}, err
			}
			skip().Run(curState)

			if !curState.InBounds(curState.Offset) || curState.Input[curState.Offset] != '[' {
				curState.Rollback(cp)
				return p.Result[PaddingStmt]{}, p.Error{
					Message: "Expected [", Expected: "[",
					Position: state.NewPositionFromState(curState),
				}
			}
			curState.Consume(1)
			skip().Run(curState)

			sizeRes, sizeErr := exprParser().Run(curState)
			if sizeErr.HasError() {
				curState.Rollback(cp)
				return p.Result[PaddingStmt]{}, sizeErr
			}
			skip().Run(curState)

			if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == ']' {
				curState.Consume(1)
			}
			skip().Run(curState)
			consumeSemicolon(curState)

			return p.NewResult(PaddingStmt{Size: sizeRes.Value}, curState,
				state.Span{Start: cp, End: curState.Save()}), p.Error{}
		},
		Label: "padding",
	}
}

func tryCatchParser() p.Parser[TryCatchStmt] {
	return p.Parser[TryCatchStmt]{
		Run: func(curState *state.State) (p.Result[TryCatchStmt], p.Error) {
			cp := curState.Save()
			_, err := keyword("try").Run(curState)
			if err.HasError() {
				return p.Result[TryCatchStmt]{}, err
			}
			skip().Run(curState)

			tryBody := parseBlock(curState)
			skip().Run(curState)

			var catchBody []Statement
			_, catchErr := keyword("catch").Run(curState)
			if !catchErr.HasError() {
				skip().Run(curState)
				catchBody = parseBlock(curState)
			}

			return p.NewResult(TryCatchStmt{Try: tryBody, Catch: catchBody}, curState,
				state.Span{Start: cp, End: curState.Save()}), p.Error{}
		},
		Label: "try-catch",
	}
}

func dollarAssignParser() p.Parser[AssignStmt] {
	return p.Parser[AssignStmt]{
		Run: func(curState *state.State) (p.Result[AssignStmt], p.Error) {
			cp := curState.Save()
			if !curState.InBounds(curState.Offset) || curState.Input[curState.Offset] != '$' {
				return p.Result[AssignStmt]{}, p.Error{
					Message: "Expected $", Expected: "$",
					Position: state.NewPositionFromState(curState),
				}
			}
			curState.Consume(1)
			skip().Run(curState)

			// Find assignment operator
			for _, op := range []string{"+=", "-=", "="} {
				if len(op) <= len(curState.Input)-curState.Offset &&
					curState.Input[curState.Offset:curState.Offset+len(op)] == op {
					curState.Consume(len(op))
					skip().Run(curState)
					valRes, valErr := exprParser().Run(curState)
					if valErr.HasError() {
						curState.Rollback(cp)
						return p.Result[AssignStmt]{}, valErr
					}
					skip().Run(curState)
					consumeSemicolon(curState)
					return p.NewResult(AssignStmt{
						Target: DollarExpr{}, Op: op, Value: valRes.Value,
					}, curState, state.Span{Start: cp, End: curState.Save()}), p.Error{}
				}
			}

			curState.Rollback(cp)
			return p.Result[AssignStmt]{}, p.Error{
				Message: "Expected assignment", Expected: "= or += or -=",
				Position: state.NewPositionFromState(curState),
			}
		},
		Label: "dollar assign",
	}
}

// --- Helpers ---

func parseBlock(curState *state.State) []Statement {
	skip().Run(curState)
	if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == '{' {
		curState.Consume(1)
		body := parseBody(curState)
		skip().Run(curState)
		if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == '}' {
			curState.Consume(1)
		}
		return body
	}
	// Single statement
	res, err := statementParser().Run(curState)
	if err.HasError() {
		return nil
	}
	return []Statement{res.Value}
}

func parenExpr(curState *state.State) (Expr, p.Error) {
	if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == '(' {
		curState.Consume(1)
		skip().Run(curState)
		res, err := exprParser().Run(curState)
		if err.HasError() {
			return nil, err
		}
		skip().Run(curState)
		if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == ')' {
			curState.Consume(1)
		}
		return res.Value, p.Error{}
	}
	// Not parenthesized — parse expression directly
	res, err := exprParser().Run(curState)
	if err.HasError() {
		return nil, err
	}
	return res.Value, p.Error{}
}

func parseAttributes(curState *state.State) []Attribute {
	var attrs []Attribute
	for {
		skip().Run(curState)
		if !curState.InBounds(curState.Offset+1) ||
			curState.Input[curState.Offset] != '[' ||
			curState.Input[curState.Offset+1] != '[' {
			break
		}
		curState.Consume(2) // [[

		for {
			skip().Run(curState)
			if !curState.InBounds(curState.Offset) {
				break
			}
			// Check for ]]
			if curState.InBounds(curState.Offset+1) &&
				curState.Input[curState.Offset] == ']' &&
				curState.Input[curState.Offset+1] == ']' {
				curState.Consume(2)
				break
			}

			nameRes, nameErr := identifier().Run(curState)
			if nameErr.HasError() {
				break
			}
			attr := Attribute{Name: nameRes.Value}

			// Check for :: in attribute name (e.g., hex::visualize)
			for curState.InBounds(curState.Offset+1) &&
				curState.Input[curState.Offset] == ':' &&
				curState.Input[curState.Offset+1] == ':' {
				curState.Consume(2)
				nextRes, nextErr := identifier().Run(curState)
				if !nextErr.HasError() {
					attr.Name += "::" + nextRes.Value
				}
			}

			skip().Run(curState)
			// Parse attribute arguments
			if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == '(' {
				curState.Consume(1)
				for {
					skip().Run(curState)
					if !curState.InBounds(curState.Offset) || curState.Input[curState.Offset] == ')' {
						break
					}
					argRes, argErr := exprParser().Run(curState)
					if argErr.HasError() {
						// Skip to ) on error
						for curState.InBounds(curState.Offset) && curState.Input[curState.Offset] != ')' {
							curState.Consume(1)
						}
						break
					}
					attr.Args = append(attr.Args, argRes.Value)
					skip().Run(curState)
					if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == ',' {
						curState.Consume(1)
					}
				}
				if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == ')' {
					curState.Consume(1)
				}
			}

			attrs = append(attrs, attr)
			skip().Run(curState)
			if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == ',' {
				curState.Consume(1)
			}
		}
	}
	return attrs
}

func parseTemplateParams(curState *state.State) []string {
	if !curState.InBounds(curState.Offset) || curState.Input[curState.Offset] != '<' {
		return nil
	}
	curState.Consume(1)
	var params []string
	for {
		skip().Run(curState)
		if !curState.InBounds(curState.Offset) || curState.Input[curState.Offset] == '>' {
			break
		}
		// Skip 'auto' keyword in template params
		autoCP := curState.Save()
		autoRes, autoErr := keyword("auto").Run(curState)
		if !autoErr.HasError() {
			_ = autoRes
			skip().Run(curState)
		} else {
			curState.Rollback(autoCP)
		}

		paramRes, paramErr := identifier().Run(curState)
		if paramErr.HasError() {
			break
		}
		params = append(params, paramRes.Value)
		skip().Run(curState)
		if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == ',' {
			curState.Consume(1)
		}
	}
	if curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == '>' {
		curState.Consume(1)
	}
	return params
}

func consumeSemicolon(curState *state.State) {
	for curState.InBounds(curState.Offset) && curState.Input[curState.Offset] == ';' {
		curState.Consume(1)
		skip().Run(curState)
	}
}

// skipIfDefBlocks consumes any #ifdef/#ifndef ... #endif blocks at the current position.
// Used between } and ; where content (like attributes) may be wrapped in conditional compilation.
// Does raw token-level skipping since the body may not contain valid items.
func skipIfDefBlocks(curState *state.State) {
	for {
		skip().Run(curState)
		if !curState.InBounds(curState.Offset) || curState.Input[curState.Offset] != '#' {
			return
		}
		// Peek to see if it's #ifdef or #ifndef
		cp := curState.Save()
		curState.Consume(1) // '#'
		idRes, idErr := identifier().Run(curState)
		if idErr.HasError() || (idRes.Value != "ifdef" && idRes.Value != "ifndef") {
			curState.Rollback(cp)
			return
		}
		// Skip everything until #endif, tracking nesting
		depth := 1
		for depth > 0 && curState.InBounds(curState.Offset) {
			// Scan for '#'
			for curState.InBounds(curState.Offset) && curState.Input[curState.Offset] != '#' {
				curState.Consume(1)
			}
			if !curState.InBounds(curState.Offset) {
				break
			}
			ppCP := curState.Save()
			curState.Consume(1) // '#'
			nestRes, nestErr := identifier().Run(curState)
			if nestErr.HasError() {
				continue
			}
			switch nestRes.Value {
			case "ifdef", "ifndef":
				depth++
			case "endif":
				depth--
			}
			_ = ppCP
		}
	}
}

func consumeCommaOrSemicolon(curState *state.State) {
	if curState.InBounds(curState.Offset) && (curState.Input[curState.Offset] == ',' || curState.Input[curState.Offset] == ';') {
		curState.Consume(1)
	}
}

// Unused import guard
var _ = strings.TrimSpace
