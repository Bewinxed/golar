package utils

import (
	"github.com/microsoft/TypeScript/tsc/pkg/ast"
	"github.com/microsoft/TypeScript/tsc/pkg/collections"
	"github.com/microsoft/TypeScript/tsc/pkg/compiler"
	"github.com/microsoft/TypeScript/tsc/pkg/core"
)

type cachedASTCompilerHost struct {
	compiler.CompilerHost
	sourceFileCache collections.SyncMap[sourceFileCacheKey, *ast.SourceFile]
}

type sourceFileCacheKey struct {
	opts       ast.SourceFileParseOptions
	text       string
	scriptKind core.ScriptKind
}

func NewCachedASTCompilerHost(baseHost compiler.CompilerHost) compiler.CompilerHost {
	return &cachedASTCompilerHost{
		CompilerHost: baseHost,
	}
}

func getSourceFileCacheKey(opts ast.SourceFileParseOptions, text string, scriptKind core.ScriptKind) sourceFileCacheKey {
	return sourceFileCacheKey{
		opts:       opts,
		text:       text,
		scriptKind: scriptKind,
	}
}

func (h *cachedASTCompilerHost) GetSourceFile(opts ast.SourceFileParseOptions) *ast.SourceFile {
	text, ok := h.FS().ReadFile(opts.FileName)
	if !ok {
		return nil
	}

	scriptKind := core.GetScriptKindFromFileName(opts.FileName)
	if scriptKind == core.ScriptKindUnknown {
		panic("Unknown script kind for file  " + opts.FileName)
	}

	key := getSourceFileCacheKey(opts, text, scriptKind)

	if cached, ok := h.sourceFileCache.Load(key); ok {
		return cached
	}

	sourceFile := h.CompilerHost.GetSourceFile(opts)
	result, _ := h.sourceFileCache.LoadOrStore(key, sourceFile)
	return result
}
