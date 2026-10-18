package explicit_anys

import (
	"github.com/auvred/golar/internal/linter/rule"

	"github.com/microsoft/TypeScript/tsc/pkg/ast"
	"github.com/microsoft/TypeScript/tsc/pkg/core"
	"github.com/microsoft/TypeScript/tsc/pkg/scanner"
)

func Rule(ctx rule.Context, opts Options) rule.Listeners {
	return rule.Listeners{
		ast.KindAnyKeyword: func(node *ast.Node) {
			isKeyofAny := isNodeWithinKeyofAny(node)

			if opts.IgnoreRestArgs && isNodeDescendantOfRestElementInFunction(node) {
				return
			}

			report := rule.NewReportForNode(ctx.SourceFile, node, "Unexpected any. Specify a different type.")

			if isKeyofAny {
				propertyKeyFix := []rule.Fix{rule.FixReplaceRange(core.NewTextRange(scanner.GetTokenPosOfNode(node.Parent, ctx.SourceFile, false), node.Parent.End()), "PropertyKey")}
				report.Suggestions = []rule.Suggestion{{
					Message: "Use `PropertyKey` instead, this is more explicit than `keyof any`.",
					Fixes:   propertyKeyFix,
				}}
				if opts.FixToUnknown {
					report.Fixes = propertyKeyFix
				}
				ctx.Report(report)
				return
			}

			report.Suggestions = []rule.Suggestion{
				{
					Message: "Use `unknown` instead, this will force you to explicitly, and safely assert the type is correct.",
					Fixes:   []rule.Fix{rule.FixReplace(ctx.SourceFile, node, "unknown")},
				},
				{
					Message: "Use `never` instead, this is useful when instantiating generic type parameters that you don't need to know the type of.",
					Fixes:   []rule.Fix{rule.FixReplace(ctx.SourceFile, node, "never")},
				},
			}

			if opts.FixToUnknown {
				report.Fixes = []rule.Fix{rule.FixReplace(ctx.SourceFile, node, "unknown")}
			}

			ctx.Report(report)
		},
	}
}

func isNodeWithinKeyofAny(node *ast.Node) bool {
	return node.Parent != nil && ast.IsTypeOperatorNode(node.Parent) && node.Parent.AsTypeOperatorNode().Operator == ast.KindKeyOfKeyword
}

func isNodeDescendantOfRestElementInFunction(node *ast.Node) bool {
	for current := node.Parent; current != nil; current = current.Parent {
		if current.Kind != ast.KindParameter {
			continue
		}

		parameter := current.AsParameterDeclaration()
		if parameter.DotDotDotToken == nil || parameter.Type == nil || !hasFunctionLikeAncestor(current.Parent) {
			return false
		}

		return isIgnoredRestParameterType(parameter.Type)
	}

	return false
}

func hasFunctionLikeAncestor(node *ast.Node) bool {
	for ; node != nil; node = node.Parent {
		if ast.IsFunctionLike(node) {
			return true
		}
	}

	return false
}

func isIgnoredRestParameterType(node *ast.TypeNode) bool {
	if node == nil {
		return false
	}

	if ast.IsArrayTypeNode(node) {
		return true
	}

	if ast.IsTypeOperatorNode(node) {
		typeOperator := node.AsTypeOperatorNode()
		return typeOperator.Operator == ast.KindReadonlyKeyword && typeOperator.Type != nil && ast.IsArrayTypeNode(typeOperator.Type)
	}

	if !ast.IsTypeReferenceNode(node) {
		return false
	}

	typeReference := node.AsTypeReferenceNode()
	if typeReference.TypeName == nil || !ast.IsIdentifier(typeReference.TypeName) {
		return false
	}

	name := typeReference.TypeName.AsIdentifier().Text
	return name == "Array" || name == "ReadonlyArray"
}
