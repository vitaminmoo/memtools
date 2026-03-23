package parser

import (
	"strconv"
	"strings"

	p "github.com/BlackBuck/pcom-go/parser"
	"github.com/BlackBuck/pcom-go/state"
)

// --- Whitespace and comments ---

// ws matches any single whitespace character (space, tab, newline, carriage return).
func ws() p.Parser[rune] {
	return p.CharWhere("whitespace", func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == '\r'
	})
}

// lineComment matches // through end of line.
func lineComment() p.Parser[string] {
	return p.Parser[string]{
		Run: func(curState *state.State) (p.Result[string], p.Error) {
			if !curState.InBounds(curState.Offset+1) ||
				curState.Input[curState.Offset] != '/' ||
				curState.Input[curState.Offset+1] != '/' {
				return p.Result[string]{}, p.Error{
					Message:  "Expected //",
					Expected: "//",
					Position: state.NewPositionFromState(curState),
				}
			}
			start := curState.Offset
			cp := curState.Save()
			// consume until newline or EOF
			end := curState.Offset
			for end < len(curState.Input) && curState.Input[end] != '\n' {
				end++
			}
			text := curState.Input[start:end]
			curState.Consume(end - start)
			return p.NewResult(text, curState, state.Span{Start: cp, End: curState.Save()}), p.Error{}
		},
		Label: "line comment",
	}
}

// blockComment matches /* ... */ including nested content.
func blockComment() p.Parser[string] {
	return p.Parser[string]{
		Run: func(curState *state.State) (p.Result[string], p.Error) {
			if !curState.InBounds(curState.Offset+1) ||
				curState.Input[curState.Offset] != '/' ||
				curState.Input[curState.Offset+1] != '*' {
				return p.Result[string]{}, p.Error{
					Message:  "Expected /*",
					Expected: "/*",
					Position: state.NewPositionFromState(curState),
				}
			}
			start := curState.Offset
			cp := curState.Save()
			end := curState.Offset + 2
			for end < len(curState.Input)-1 {
				if curState.Input[end] == '*' && curState.Input[end+1] == '/' {
					end += 2
					text := curState.Input[start:end]
					curState.Consume(end - start)
					return p.NewResult(text, curState, state.Span{Start: cp, End: curState.Save()}), p.Error{}
				}
				end++
			}
			return p.Result[string]{}, p.Error{
				Message:  "Unterminated block comment",
				Expected: "*/",
				Position: state.NewPositionFromState(curState),
			}
		},
		Label: "block comment",
	}
}

// skip consumes whitespace and comments, returning nothing useful.
func skip() p.Parser[[]string] {
	item := p.Or("ws or comment",
		p.Map("ws->str", ws(), func(r rune) string { return string(r) }),
		lineComment(),
		blockComment(),
	)
	return p.Many0("skip", item)
}

// lexeme wraps a parser to consume trailing whitespace/comments.
func lexeme[T any](parser p.Parser[T]) p.Parser[T] {
	return p.Parser[T]{
		Run: func(curState *state.State) (p.Result[T], p.Error) {
			cp := curState.Save()
			res, err := parser.Run(curState)
			if err.HasError() {
				curState.Rollback(cp)
				return res, err
			}
			// consume trailing whitespace/comments
			skip().Run(res.NextState)
			return p.Result[T]{
				Value:     res.Value,
				NextState: res.NextState,
				Span:      res.Span,
			}, p.Error{}
		},
		Label: parser.Label,
	}
}

// --- Identifiers and keywords ---

var keywords = map[string]bool{
	"struct": true, "union": true, "enum": true, "bitfield": true,
	"using": true, "fn": true, "namespace": true,
	"if": true, "else": true, "while": true, "for": true,
	"match": true, "return": true, "break": true, "continue": true,
	"try": true, "catch": true,
	"true": true, "false": true,
	"padding": true, "sizeof": true, "addressof": true,
	"le": true, "be": true,
	"in": true, "out": true, "ref": true, "auto": true,
	"this": true, "parent": true, "null": true,
	"import": true,
}

// identifier parses a C-style identifier (letter or underscore, then alphanumeric or underscore).
func identifier() p.Parser[string] {
	return p.Parser[string]{
		Run: func(curState *state.State) (p.Result[string], p.Error) {
			if !curState.InBounds(curState.Offset) {
				return p.Result[string]{}, p.Error{
					Message:  "Expected identifier",
					Expected: "identifier",
					Got:      "EOF",
					Position: state.NewPositionFromState(curState),
				}
			}
			c := curState.Input[curState.Offset]
			if !isIdentStart(c) {
				return p.Result[string]{}, p.Error{
					Message:  "Expected identifier",
					Expected: "identifier",
					Got:      string(c),
					Position: state.NewPositionFromState(curState),
				}
			}
			cp := curState.Save()
			end := curState.Offset
			for end < len(curState.Input) && isIdentCont(curState.Input[end]) {
				end++
			}
			name := curState.Input[curState.Offset:end]
			curState.Consume(end - curState.Offset)
			return p.NewResult(name, curState, state.Span{Start: cp, End: curState.Save()}), p.Error{}
		},
		Label: "identifier",
	}
}

// nonKeywordIdent parses an identifier that is not a keyword.
func nonKeywordIdent() p.Parser[string] {
	return p.Parser[string]{
		Run: func(curState *state.State) (p.Result[string], p.Error) {
			cp := curState.Save()
			res, err := identifier().Run(curState)
			if err.HasError() {
				return res, err
			}
			if keywords[res.Value] {
				curState.Rollback(cp)
				return p.Result[string]{}, p.Error{
					Message:  "Identifier is a keyword",
					Expected: "non-keyword identifier",
					Got:      res.Value,
					Position: state.NewPositionFromState(curState),
				}
			}
			return res, p.Error{}
		},
		Label: "non-keyword identifier",
	}
}

// keyword parses a specific keyword and ensures it's not followed by an ident character.
func keyword(kw string) p.Parser[string] {
	return p.Parser[string]{
		Run: func(curState *state.State) (p.Result[string], p.Error) {
			cp := curState.Save()
			res, err := identifier().Run(curState)
			if err.HasError() {
				return p.Result[string]{}, p.Error{
					Message:  "Expected keyword " + kw,
					Expected: kw,
					Got:      err.Got,
					Position: state.NewPositionFromState(curState),
				}
			}
			if res.Value != kw {
				curState.Rollback(cp)
				return p.Result[string]{}, p.Error{
					Message:  "Expected keyword " + kw,
					Expected: kw,
					Got:      res.Value,
					Position: state.NewPositionFromState(curState),
				}
			}
			return res, p.Error{}
		},
		Label: "keyword " + kw,
	}
}

func isIdentStart(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

func isIdentCont(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
}

// --- Number literals ---

// numberLit parses integer literals: decimal, hex (0x), binary (0b), octal (0o).
func numberLit() p.Parser[NumberLit] {
	return p.Parser[NumberLit]{
		Run: func(curState *state.State) (p.Result[NumberLit], p.Error) {
			if !curState.InBounds(curState.Offset) {
				return p.Result[NumberLit]{}, p.Error{
					Message: "Expected number", Expected: "number", Got: "EOF",
					Position: state.NewPositionFromState(curState),
				}
			}
			cp := curState.Save()
			start := curState.Offset
			end := start

			// Check for negative sign
			if end < len(curState.Input) && curState.Input[end] == '-' {
				end++
			}

			if !curState.InBounds(end) || !isDigit(curState.Input[end]) {
				// Special case: could be 0x, 0b, 0o prefix
				if end < len(curState.Input) && curState.Input[end] == '0' {
					// handled below
				} else {
					return p.Result[NumberLit]{}, p.Error{
						Message: "Expected number", Expected: "number",
						Got:      string(curState.Input[start]),
						Position: state.NewPositionFromState(curState),
					}
				}
			}

			// Determine base
			digitStart := end
			if end < len(curState.Input)-1 && curState.Input[end] == '0' {
				next := curState.Input[end+1]
				switch next {
				case 'x', 'X':
					end += 2
					for end < len(curState.Input) && (isHexDigit(curState.Input[end]) || curState.Input[end] == '\'') {
						end++
					}
				case 'b', 'B':
					end += 2
					for end < len(curState.Input) && (curState.Input[end] == '0' || curState.Input[end] == '1' || curState.Input[end] == '\'') {
						end++
					}
				case 'o', 'O':
					end += 2
					for end < len(curState.Input) && (curState.Input[end] >= '0' && curState.Input[end] <= '7' || curState.Input[end] == '\'') {
						end++
					}
				default:
					for end < len(curState.Input) && (isDigit(curState.Input[end]) || curState.Input[end] == '\'') {
						end++
					}
				}
			} else {
				for end < len(curState.Input) && (isDigit(curState.Input[end]) || curState.Input[end] == '\'') {
					end++
				}
			}

			if end == digitStart && curState.Input[start] != '0' {
				return p.Result[NumberLit]{}, p.Error{
					Message: "Expected number", Expected: "number",
					Position: state.NewPositionFromState(curState),
				}
			}

			raw := curState.Input[start:end]
			// Check for trailing U suffix (unsigned)
			if end < len(curState.Input) && (curState.Input[end] == 'U' || curState.Input[end] == 'u') {
				end++
			}

			val, err := parseIntLit(raw)
			if err != nil {
				return p.Result[NumberLit]{}, p.Error{
					Message:  "Invalid number literal: " + raw,
					Expected: "valid number",
					Got:      raw,
					Position: state.NewPositionFromState(curState),
				}
			}

			curState.Consume(end - start)
			return p.NewResult(
				NumberLit{Value: val, Raw: raw},
				curState,
				state.Span{Start: cp, End: curState.Save()},
			), p.Error{}
		},
		Label: "number",
	}
}

func parseIntLit(s string) (int64, error) {
	// Strip digit separators
	s = strings.ReplaceAll(s, "'", "")
	neg := false
	if len(s) > 0 && s[0] == '-' {
		neg = true
		s = s[1:]
	}
	var val uint64
	var err error
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		val, err = strconv.ParseUint(s[2:], 16, 64)
	} else if strings.HasPrefix(s, "0b") || strings.HasPrefix(s, "0B") {
		val, err = strconv.ParseUint(s[2:], 2, 64)
	} else if strings.HasPrefix(s, "0o") || strings.HasPrefix(s, "0O") {
		val, err = strconv.ParseUint(s[2:], 8, 64)
	} else {
		val, err = strconv.ParseUint(s, 10, 64)
	}
	result := int64(val)
	if neg {
		result = -result
	}
	return result, err
}

// floatLit parses float literals like 1.414F or 3.14159.
func floatLit() p.Parser[FloatLit] {
	return p.Parser[FloatLit]{
		Run: func(curState *state.State) (p.Result[FloatLit], p.Error) {
			if !curState.InBounds(curState.Offset) {
				return p.Result[FloatLit]{}, p.Error{
					Message: "Expected float", Expected: "float", Got: "EOF",
					Position: state.NewPositionFromState(curState),
				}
			}
			cp := curState.Save()
			start := curState.Offset
			end := start

			// optional negative
			if end < len(curState.Input) && curState.Input[end] == '-' {
				end++
			}

			// digits before dot
			for end < len(curState.Input) && isDigit(curState.Input[end]) {
				end++
			}

			// require dot
			if end >= len(curState.Input) || curState.Input[end] != '.' {
				return p.Result[FloatLit]{}, p.Error{
					Message: "Expected float", Expected: "float with decimal point",
					Position: state.NewPositionFromState(curState),
				}
			}
			end++ // consume dot

			// digits after dot
			for end < len(curState.Input) && isDigit(curState.Input[end]) {
				end++
			}

			raw := curState.Input[start:end]
			// optional F/D suffix
			if end < len(curState.Input) && (curState.Input[end] == 'F' || curState.Input[end] == 'f' ||
				curState.Input[end] == 'D' || curState.Input[end] == 'd') {
				end++
			}

			val, parseErr := strconv.ParseFloat(raw, 64)
			if parseErr != nil {
				return p.Result[FloatLit]{}, p.Error{
					Message: "Invalid float: " + raw, Expected: "valid float",
					Position: state.NewPositionFromState(curState),
				}
			}

			curState.Consume(end - start)
			return p.NewResult(FloatLit{Value: val}, curState,
				state.Span{Start: cp, End: curState.Save()}), p.Error{}
		},
		Label: "float",
	}
}

// --- String and char literals ---

// stringLit parses "..." with escape sequences.
func stringLit() p.Parser[StringLit] {
	return p.Parser[StringLit]{
		Run: func(curState *state.State) (p.Result[StringLit], p.Error) {
			if !curState.InBounds(curState.Offset) || curState.Input[curState.Offset] != '"' {
				return p.Result[StringLit]{}, p.Error{
					Message: "Expected string", Expected: "\"",
					Position: state.NewPositionFromState(curState),
				}
			}
			cp := curState.Save()
			end := curState.Offset + 1
			var b strings.Builder
			for end < len(curState.Input) {
				c := curState.Input[end]
				if c == '\\' && end+1 < len(curState.Input) {
					end++
					switch curState.Input[end] {
					case 'n':
						b.WriteByte('\n')
					case 'r':
						b.WriteByte('\r')
					case 't':
						b.WriteByte('\t')
					case '\\':
						b.WriteByte('\\')
					case '"':
						b.WriteByte('"')
					case '0':
						b.WriteByte(0)
					case 'x':
						if end+2 < len(curState.Input) {
							hex := curState.Input[end+1 : end+3]
							v, err := strconv.ParseUint(hex, 16, 8)
							if err == nil {
								b.WriteByte(byte(v))
								end += 2
							} else {
								b.WriteByte('x')
							}
						}
					default:
						b.WriteByte(curState.Input[end])
					}
					end++
					continue
				}
				if c == '"' {
					end++
					curState.Consume(end - curState.Offset)
					return p.NewResult(StringLit{Value: b.String()}, curState,
						state.Span{Start: cp, End: curState.Save()}), p.Error{}
				}
				b.WriteByte(c)
				end++
			}
			return p.Result[StringLit]{}, p.Error{
				Message: "Unterminated string", Expected: "\"",
				Position: state.NewPositionFromState(curState),
			}
		},
		Label: "string",
	}
}

// charLit parses 'x' character literals.
func charLit() p.Parser[CharLit] {
	return p.Parser[CharLit]{
		Run: func(curState *state.State) (p.Result[CharLit], p.Error) {
			if !curState.InBounds(curState.Offset) || curState.Input[curState.Offset] != '\'' {
				return p.Result[CharLit]{}, p.Error{
					Message: "Expected char", Expected: "'",
					Position: state.NewPositionFromState(curState),
				}
			}
			cp := curState.Save()
			end := curState.Offset + 1
			var ch rune
			if end < len(curState.Input) && curState.Input[end] == '\\' {
				end++
				if end < len(curState.Input) {
					switch curState.Input[end] {
					case 'n':
						ch = '\n'
					case 'r':
						ch = '\r'
					case 't':
						ch = '\t'
					case '\\':
						ch = '\\'
					case '\'':
						ch = '\''
					case '0':
						ch = 0
					case 'x':
						if end+2 < len(curState.Input) {
							hex := curState.Input[end+1 : end+3]
							v, parseErr := strconv.ParseUint(hex, 16, 8)
							if parseErr == nil {
								ch = rune(v)
								end += 2
							} else {
								ch = 'x'
							}
						} else {
							ch = 'x'
						}
					default:
						ch = rune(curState.Input[end])
					}
					end++
				}
			} else if end < len(curState.Input) {
				ch = rune(curState.Input[end])
				end++
			}
			if end < len(curState.Input) && curState.Input[end] == '\'' {
				end++
				curState.Consume(end - curState.Offset)
				return p.NewResult(CharLit{Value: ch}, curState,
					state.Span{Start: cp, End: curState.Save()}), p.Error{}
			}
			return p.Result[CharLit]{}, p.Error{
				Message: "Unterminated char literal", Expected: "'",
				Position: state.NewPositionFromState(curState),
			}
		},
		Label: "char",
	}
}

// --- Punctuation helpers ---

func sym(s string) p.Parser[string] {
	return lexeme(p.StringParser(s, s))
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func isHexDigit(c byte) bool {
	return isDigit(c) || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}
