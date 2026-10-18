package rule

import (
	"github.com/auvred/golar/internal/utils"
	"github.com/microsoft/TypeScript/tsc/pkg/ast"
	"github.com/microsoft/TypeScript/tsc/pkg/checker"
	"github.com/microsoft/TypeScript/tsc/pkg/compiler"
	"github.com/microsoft/TypeScript/tsc/pkg/core"
)

type Report struct {
	Range       core.TextRange `json:"range"`
	Message     string         `json:"message"`
	Fixes       []Fix          `json:"fixes"`
	Suggestions []Suggestion   `json:"suggestions"`
}

func (r Report) GetFixes() []Fix {
	return r.Fixes
}

type Fix struct {
	Text  string         `json:"text"`
	Range core.TextRange `json:"range"`
}

func FixInsertBefore(file *ast.SourceFile, node *ast.Node, text string) Fix {
	trimmed := utils.TrimNodeTextRange(file, node)
	return Fix{
		Text:  text,
		Range: trimmed.WithEnd(trimmed.Pos()),
	}
}
func FixInsertAfter(node *ast.Node, text string) Fix {
	return Fix{
		Text:  text,
		Range: node.Loc.WithPos(node.End()),
	}
}
func FixReplace(file *ast.SourceFile, node *ast.Node, text string) Fix {
	return FixReplaceRange(utils.TrimNodeTextRange(file, node), text)
}
func FixReplaceRange(textRange core.TextRange, text string) Fix {
	return Fix{
		Text:  text,
		Range: textRange,
	}
}
func FixRemove(file *ast.SourceFile, node *ast.Node) Fix {
	return FixReplace(file, node, "")
}
func FixRemoveRange(textRange core.TextRange) Fix {
	return FixReplaceRange(textRange, "")
}

type Suggestion struct {
	Message string `json:"message"`
	Fixes   []Fix  `json:"fixes"`
}

func (s Suggestion) GetFixes() []Fix {
	return s.Fixes
}

func NewReportForNode(sourceFile *ast.SourceFile, node *ast.Node, message string) Report {
	return Report{
		Range:   utils.TrimNodeTextRange(sourceFile, node),
		Message: message,
	}
}

type Context struct {
	Program     *compiler.Program
	TypeChecker *checker.Checker
	SourceFile  *ast.SourceFile

	Report func(report Report)
}

type Listeners map[ast.Kind](func(node *ast.Node))

const (
	tokenKindLast ast.Kind = iota * 1000
	tokenKindOnExit
	tokenKindInBindingPosition
	tokenKindOnExitInBindingPosition
	tokenKindNotInBindingPosition
	tokenKindOnExitNotInBindingPosition
)

func ListenerOnExit(kind ast.Kind) ast.Kind {
	return kind + 1000
}

func ListenerInBindingPosition(kind ast.Kind) ast.Kind {
	return kind + tokenKindInBindingPosition
}
func ListenerNotInBindingPosition(kind ast.Kind) ast.Kind {
	return kind + tokenKindNotInBindingPosition
}
