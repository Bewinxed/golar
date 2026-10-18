package utils

import (
	"github.com/microsoft/TypeScript/tsc/pkg/ast"
	"github.com/microsoft/TypeScript/tsc/pkg/compiler"
	"github.com/microsoft/TypeScript/tsc/pkg/tspath"
)

type IndexedCompilerHost struct {
	compiler.CompilerHost
	currentDirectory tspath.RootedDirectoryPath
}

func NewIndexedCompilerHost(base compiler.CompilerHost, currentDirectory tspath.RootedDirectoryPath) compiler.CompilerHost {
	return &IndexedCompilerHost{CompilerHost: base, currentDirectory: currentDirectory}
}

func (h *IndexedCompilerHost) GetCurrentDirectory() tspath.RootedDirectoryPath {
	return h.currentDirectory
}

func (h *IndexedCompilerHost) GetSourceFile(opts ast.SourceFileParseOptions) *ast.SourceFile {
	sourceFile := h.CompilerHost.GetSourceFile(opts)
	stampSourceFileNodeIDs(sourceFile)
	return sourceFile
}

func stampSourceFileNodeIDs(sourceFile *ast.SourceFile) {
	if sourceFile == nil {
		return
	}

	root := sourceFile.AsNode()
	if root.Id == 1 && root.SourceFileId == sourceFile.Id {
		return
	}

	var nodeIndex uint32 = 1
	root.Id = nodeIndex
	root.SourceFileId = sourceFile.Id

	visitor := &ast.NodeVisitor{
		Hooks: ast.NodeVisitorHooks{
			VisitNodes: func(nodeList *ast.NodeList, visitor *ast.NodeVisitor) *ast.NodeList {
				if nodeList == nil || len(nodeList.Nodes) == 0 {
					return nodeList
				}

				nodeIndex++
				visitor.VisitSlice(nodeList.Nodes)
				return nodeList
			},
			VisitModifiers: func(modifiers *ast.ModifierList, visitor *ast.NodeVisitor) *ast.ModifierList {
				if modifiers != nil && len(modifiers.Nodes) > 0 {
					visitor.Hooks.VisitNodes(&modifiers.NodeList, visitor)
				}
				return modifiers
			},
		},
	}

	visitor.Visit = func(node *ast.Node) *ast.Node {
		nodeIndex++
		node.Id = nodeIndex
		node.SourceFileId = sourceFile.Id

		visitor.VisitEachChild(node)
		for _, jsdoc := range node.JSDoc(sourceFile) {
			visitor.Visit(jsdoc)
		}

		return node
	}

	visitor.VisitEachChild(root)
	for _, jsdoc := range root.JSDoc(sourceFile) {
		visitor.Visit(jsdoc)
	}
}
