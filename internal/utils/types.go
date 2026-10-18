package utils

import (
	"github.com/microsoft/TypeScript/tsc/pkg/ast"
	"github.com/microsoft/TypeScript/tsc/pkg/checker"
	"github.com/microsoft/TypeScript/tsc/pkg/core"
)

func IsTypeFlagSet(t *checker.Type, flag checker.TypeFlags) bool {
	return t != nil && t.Flags()&flag != 0
}

func IsSymbolFlagSet(symbol *ast.Symbol, flag ast.SymbolFlags) bool {
	return symbol != nil && symbol.Flags()&flag != 0
}

func UnionTypeParts(t *checker.Type) []*checker.Type {
	if IsTypeFlagSet(t, checker.TypeFlagsUnion) {
		return t.AsUnionOrIntersectionType().Types()
	}
	// it doesn't allocate when inlined
	return []*checker.Type{t}
}

func IntersectionTypeParts(t *checker.Type) []*checker.Type {
	if IsTypeFlagSet(t, checker.TypeFlagsIntersection) {
		return t.AsUnionOrIntersectionType().Types()
	}
	// it doesn't allocate when inlined
	return []*checker.Type{t}
}

func TypeRecurser(t *checker.Type, fn func(t *checker.Type) bool) bool {
	if t == nil {
		return false
	}
	if IsTypeFlagSet(t, checker.TypeFlagsUnionOrIntersection) {
		return core.Some(t.AsUnionOrIntersectionType().Types(), func(subtype *checker.Type) bool {
			return TypeRecurser(subtype, fn)
		})
	}
	return fn(t)
}

func IsIntrinsicErrorType(t *checker.Type) bool {
	return IsTypeFlagSet(t, checker.TypeFlagsIntrinsic) && t.AsIntrinsicType().IntrinsicName() == "error"
}

func GetConstrainedType(typeChecker *checker.Checker, t *checker.Type) *checker.Type {
	if IsTypeFlagSet(t, checker.TypeFlagsTypeParameter) {
		res := typeChecker.GetBaseConstraintOfType(t)
		if res == nil {
			return typeChecker.GetUnknownType()
		}
		return res
	}
	return t
}

func HasBaseType(
	typeChecker *checker.Checker,
	t *checker.Type,
	predicate func(subType *checker.Type) bool,
) bool {
	if predicate(t) {
		return true
	}

	if IsTypeFlagSet(t, checker.TypeFlagsIntersection) {
		return core.Some(IntersectionTypeParts(t), func(t *checker.Type) bool {
			return HasBaseType(typeChecker, t, predicate)
		})
	}
	if IsTypeFlagSet(t, checker.TypeFlagsUnion) {
		return core.Every(UnionTypeParts(t), func(t *checker.Type) bool {
			return HasBaseType(typeChecker, t, predicate)
		})
	}

	t = GetConstrainedType(typeChecker, t)

	if predicate(t) {
		return true
	}

	symbol := t.Symbol()
	if IsSymbolFlagSet(symbol, ast.SymbolFlagsClass|ast.SymbolFlagsInterface) {
		declaredType := typeChecker.GetDeclaredTypeOfSymbol(symbol)
		for _, baseType := range typeChecker.GetBaseTypes(declaredType) {
			if HasBaseType(typeChecker, baseType, predicate) {
				return true
			}
		}
	}
	return false
}
