package linter

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/auvred/golar/internal/linter/rule"

	"github.com/microsoft/TypeScript/tsc/pkg/ast"
	"github.com/microsoft/TypeScript/tsc/pkg/bundled"
	"github.com/microsoft/TypeScript/tsc/pkg/checker"
	"github.com/microsoft/TypeScript/tsc/pkg/compiler"
	"github.com/microsoft/TypeScript/tsc/pkg/core"
	"github.com/microsoft/TypeScript/tsc/pkg/tsoptions"
	"github.com/microsoft/TypeScript/tsc/pkg/tspath"
	"github.com/microsoft/TypeScript/tsc/pkg/vfs"
	"github.com/microsoft/TypeScript/tsc/pkg/vfs/vfstest"
)

func ConfigureRule(ruleName string, options []byte, program *compiler.Program, typeChecker *checker.Checker, sourceFile *ast.SourceFile, onReport func(rule.Report)) rule.Listeners {
	ctx := rule.Context{
		Program:     program,
		TypeChecker: typeChecker,
		SourceFile:  sourceFile,
		Report:      onReport,
	}

	return setupRule(ruleName, ctx, options)
}

func VisitSourceFile(file *ast.SourceFile, visitor func(ast.Kind, *ast.Node)) {
	var childVisitor ast.Visitor
	var patternVisitor func(node *ast.Node)
	patternVisitor = func(node *ast.Node) {
		visitor(node.Kind, node)
		kind := rule.ListenerInBindingPosition(node.Kind)
		visitor(kind, node)

		switch node.Kind {
		case ast.KindArrayLiteralExpression:
			for _, element := range node.AsArrayLiteralExpression().Elements.Nodes {
				patternVisitor(element)
			}
		case ast.KindObjectLiteralExpression:
			for _, property := range node.AsObjectLiteralExpression().Properties.Nodes {
				patternVisitor(property)
			}
		case ast.KindSpreadElement, ast.KindSpreadAssignment:
			patternVisitor(node.Expression())
		case ast.KindPropertyAssignment:
			patternVisitor(node.Initializer())
		default:
			node.ForEachChild(childVisitor)
		}

		visitor(rule.ListenerOnExit(kind), node)
		visitor(rule.ListenerOnExit(node.Kind), node)
	}
	childVisitor = func(node *ast.Node) bool {
		visitor(node.Kind, node)

		switch node.Kind {
		case ast.KindArrayLiteralExpression, ast.KindObjectLiteralExpression:
			kind := rule.ListenerNotInBindingPosition(node.Kind)
			visitor(kind, node)
			node.ForEachChild(childVisitor)
			visitor(rule.ListenerOnExit(kind), node)
		default:
			if ast.IsAssignmentExpression(node, true) {
				expr := node.AsBinaryExpression()
				patternVisitor(expr.Left)
				childVisitor(expr.OperatorToken)
				childVisitor(expr.Right)
			} else {
				node.ForEachChild(childVisitor)
			}
		}

		visitor(rule.ListenerOnExit(node.Kind), node)

		return false
	}
	file.Node.ForEachChild(childVisitor)
}

func RuleTesterLint(files string, fileName string, ruleName string, options string) string {
	fileMap := map[string]string{}
	if err := json.Unmarshal([]byte(files), &fileMap); err != nil {
		panic(err)
	}

	fs := bundled.WrapFS(vfstest.FromMap(fileMap, tspath.CaseSensitive))
	cwd := "/"
	configFileName := "/tsconfig.json"
	host := compiler.NewCompilerHost(fs, bundled.LibPath(), nil, nil, nil)

	program, err := createProgramFromOpts(fs, cwd, configFileName, host)
	if err != nil {
		panic(err)
	}

	sourceFile := program.GetSourceFile(tspath.RootedFilePathFromNormalized(fileName))
	typeChecker, done := program.GetTypeCheckerForFile(context.Background(), sourceFile)
	defer done()

	reports := []rule.Report{}

	listeners := ConfigureRule(ruleName, []byte(options), program, typeChecker, sourceFile, func(report rule.Report) {
		reports = append(reports, report)
	})

	VisitSourceFile(sourceFile, func(kind ast.Kind, node *ast.Node) {
		if listener, ok := listeners[kind]; ok {
			listener(node)
		}
	})

	slices.SortFunc(reports, func(left rule.Report, right rule.Report) int {
		if left.Range.Pos() != right.Range.Pos() {
			return left.Range.Pos() - right.Range.Pos()
		}
		return left.Range.End() - right.Range.End()
	})

	snapshot := fileMap[fileName]
	for i := len(reports) - 1; i >= 0; i-- {
		snapshot = createReportSnapshotAt(snapshot, reports[i])
	}

	fixedOutput, _, fixed := ApplyRuleFixes(fileMap[fileName], reports)

	type lintSuggestionResult struct {
		Message string `json:"message"`
		Output  string `json:"output"`
	}

	suggestionResults := []lintSuggestionResult{}
	for _, report := range reports {
		for _, suggestion := range report.Suggestions {
			suggestedOutput, _, suggestionFixed := ApplyRuleFixes(fileMap[fileName], []rule.Suggestion{suggestion})
			if !suggestionFixed {
				continue
			}

			suggestionResults = append(suggestionResults, lintSuggestionResult{
				Message: suggestion.Message,
				Output:  suggestedOutput,
			})
		}
	}

	type lintResult struct {
		Snapshot    string                 `json:"snapshot"`
		Output      string                 `json:"output,omitempty"`
		Suggestions []lintSuggestionResult `json:"suggestions,omitempty"`
	}

	result := lintResult{
		Snapshot: snapshot,
	}
	if fixed {
		result.Output = fixedOutput
	}
	if len(suggestionResults) > 0 {
		result.Suggestions = suggestionResults
	}

	res, err := json.Marshal(result)
	if err != nil {
		panic(err)
	}
	return string(res)
}

func createReportSnapshotAt(sourceText string, report rule.Report) string {
	pos := report.Range.Pos()
	end := report.Range.End()
	if end < pos {
		end = pos
	}

	pos = min(max(0, pos), len(sourceText))
	end = min(max(0, end), len(sourceText))

	lineStartIndex := strings.LastIndex(sourceText[:pos], "\n") + 1
	lineEndIndex := len(sourceText)
	if nextLineBreak := strings.Index(sourceText[end:], "\n"); nextLineBreak >= 0 {
		lineEndIndex = end + nextLineBreak
	}

	before := sourceText[:lineStartIndex]
	between := sourceText[lineStartIndex:lineEndIndex]
	after := sourceText[lineEndIndex:]

	lines := strings.Split(between, "\n")
	output := make([]string, 0, len(lines)*3)
	currentLineStart := lineStartIndex
	messageInserted := false
	for _, line := range lines {
		currentLineEnd := currentLineStart + len(line)
		markerStart := max(0, min(len(line), pos-currentLineStart))
		markerEnd := max(markerStart+1, min(len(line), end-currentLineStart))
		indentEnd, firstNonWhitespace, lastNonWhitespaceEnd := getLineNonWhitespaceBounds(line)
		hasVisibleContent := firstNonWhitespace >= 0

		if hasVisibleContent {
			markerEnd = min(markerEnd, lastNonWhitespaceEnd)

			if pos <= currentLineStart {
				markerStart = max(markerStart, firstNonWhitespace)
			}

			fullyCoveredLine := pos <= currentLineStart && end >= currentLineEnd
			if fullyCoveredLine {
				markerStart = firstNonWhitespace
				markerEnd = lastNonWhitespaceEnd
			}

			if markerEnd <= markerStart && markerStart < lastNonWhitespaceEnd {
				markerEnd = min(lastNonWhitespaceEnd, markerStart+1)
			}
		}

		output = append(output, line)
		markerLine := ""
		markerPrefix := ""
		if !hasVisibleContent || markerStart >= len(line) || markerEnd <= markerStart {
			output = append(output, markerLine)
		} else {
			markerPrefix = createSnapshotMarkerPrefix(line, indentEnd, markerStart)
			markerLine = markerPrefix + strings.Repeat("~", markerEnd-markerStart)
			output = append(output, markerLine)
		}

		if !messageInserted && markerLine != "" {
			for _, messageLine := range strings.Split(report.Message, "\n") {
				output = append(output, markerPrefix+messageLine)
			}
			messageInserted = true
		}

		currentLineStart = currentLineEnd + 1
	}

	if !messageInserted && len(output) > 0 {
		output = append(output, report.Message)
	}

	return before + strings.Join(output, "\n") + after
}

func getLineNonWhitespaceBounds(line string) (indentEnd int, firstNonWhitespace int, lastNonWhitespaceEnd int) {
	indentEnd = len(line)
	firstNonWhitespace = -1

	for i, r := range line {
		if !unicode.IsSpace(r) {
			if firstNonWhitespace < 0 {
				indentEnd = i
				firstNonWhitespace = i
			}
			lastNonWhitespaceEnd = i + utf8.RuneLen(r)
		}
	}

	return indentEnd, firstNonWhitespace, lastNonWhitespaceEnd
}

func createSnapshotMarkerPrefix(line string, indentEnd int, markerStart int) string {
	indentEnd = min(indentEnd, len(line))
	if markerStart <= indentEnd {
		return line[:markerStart]
	}

	return line[:indentEnd] + strings.Repeat(" ", markerStart-indentEnd)
}

func createProgramFromOpts(fs vfs.FS, cwd string, tsconfigPath string, host compiler.CompilerHost) (*compiler.Program, error) {
	configParseResult, _ := tsoptions.GetParsedCommandLineOfConfigFile(
		tspath.RootedFilePathFromNormalized(tsconfigPath),
		&core.CompilerOptions{},
		nil,
		fs,
		nil,
	)

	opts := compiler.ProgramOptions{
		Config: configParseResult,
		// TODO:
		SingleThreaded: core.TSTrue,
		Host:           host,
	}
	program := compiler.NewProgram(opts)

	diagnostics := program.GetSyntacticDiagnostics(context.Background(), nil)
	if len(diagnostics) != 0 {
		return nil, fmt.Errorf("found %v syntactic errors\n", len(diagnostics))
	}

	return program, nil
}
