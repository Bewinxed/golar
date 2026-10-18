package tscodegenplugin

import (
	"encoding/binary"
	"unsafe"

	"github.com/auvred/golar/internal/mapping"
	"github.com/auvred/golar/plugin"
	"github.com/microsoft/TypeScript/tsc/pkg/core"
)

type ServiceCodeError struct {
	Message string
	Loc     core.TextRange
}

type CreateServiceCodeRequest struct {
	Cwd            string
	ConfigFileName string
	FileName       string
	SourceText     string
}

type CreateServiceCodeResponse struct {
	Errors              []ServiceCodeError
	ServiceText         string
	ScriptKind          core.ScriptKind
	Mappings            []mapping.Mapping
	IgnoreMappings      []mapping.IgnoreDirectiveMapping
	ExpectErrorMappings []mapping.ExpectErrorDirectiveMapping
	DeclarationFile     bool
	// For Volar.js compat
	IgnoreNotMappedDiagnostics bool
}

type Plugin interface {
	CreateServiceCode(req CreateServiceCodeRequest) CreateServiceCodeResponse
	Extensions() []plugin.FileExtension
}

func encodedLenCreateServiceCodeRequest(req CreateServiceCodeRequest) int {
	return 4 + len(req.FileName) + 4 + len(req.ConfigFileName) + 4 + len(req.FileName) + 4 + len(req.SourceText)
}

func encodeCreateServiceCodeRequest(buf []byte, req CreateServiceCodeRequest) {
	offset := 0
	binary.LittleEndian.PutUint32(buf[offset:], uint32(len(req.Cwd)))
	offset += 4
	copy(buf[offset:], req.Cwd)
	offset += len(req.Cwd)
	binary.LittleEndian.PutUint32(buf[offset:], uint32(len(req.ConfigFileName)))
	offset += 4
	copy(buf[offset:], req.ConfigFileName)
	offset += len(req.ConfigFileName)
	binary.LittleEndian.PutUint32(buf[offset:], uint32(len(req.FileName)))
	offset += 4
	copy(buf[offset:], req.FileName)
	offset += len(req.FileName)
	binary.LittleEndian.PutUint32(buf[offset:], uint32(len(req.SourceText)))
	offset += 4
	copy(buf[offset:], req.SourceText)
	offset += len(req.SourceText)
}

func decodeCreateServiceCodeResponse(buf []byte) (response CreateServiceCodeResponse) {
	offset := uint32(0)

	properties := plugin.ServiceCodeProperties(buf[offset])
	offset += 1
	if properties&plugin.ServiceCodePropertiesError != 0 {
		errorsCount := binary.LittleEndian.Uint32(buf[offset:])
		offset += 4
		response.Errors = make([]ServiceCodeError, errorsCount)
		for i := range errorsCount {
			messageLen := binary.LittleEndian.Uint32(buf[offset:])
			offset += 4
			response.Errors[i].Message = string(buf[offset : offset+messageLen])
			offset += messageLen
			start := binary.LittleEndian.Uint32(buf[offset:])
			offset += 4
			end := binary.LittleEndian.Uint32(buf[offset:])
			offset += 4
			response.Errors[i].Loc = core.NewTextRange(int(start), int(end))
		}
		return
	}

	scriptKind := plugin.ScriptKind(buf[offset])
	offset += 1

	switch scriptKind {
	case plugin.ScriptKindJS:
		response.ScriptKind = core.ScriptKindJS
	case plugin.ScriptKindJSX:
		response.ScriptKind = core.ScriptKindJSX
	case plugin.ScriptKindTS:
		response.ScriptKind = core.ScriptKindTS
	case plugin.ScriptKindTSX:
		response.ScriptKind = core.ScriptKindTSX
	}

	response.DeclarationFile = properties&plugin.ServiceCodePropertiesDeclarationFile != 0
	response.IgnoreNotMappedDiagnostics = properties&plugin.ServiceCodePropertiesIgnoreNotMappedDiagnostics != 0

	serviceTextLen := binary.LittleEndian.Uint32(buf[offset:])
	offset += 4
	response.ServiceText = string(buf[offset : offset+serviceTextLen])
	offset += serviceTextLen

	mappingsCount := binary.LittleEndian.Uint32(buf[offset:])
	offset += 4
	response.Mappings = make([]mapping.Mapping, mappingsCount)
	for i := range mappingsCount {
		response.Mappings[i].SourceOffset = binary.LittleEndian.Uint32(buf[offset:])
		offset += 4
		response.Mappings[i].ServiceOffset = binary.LittleEndian.Uint32(buf[offset:])
		offset += 4
		response.Mappings[i].SourceLength = binary.LittleEndian.Uint32(buf[offset:])
		offset += 4
		response.Mappings[i].ServiceLength = binary.LittleEndian.Uint32(buf[offset:])
		offset += 4
		suppressedDiagnosticsCount := binary.LittleEndian.Uint32(buf[offset:])
		offset += 4
		if suppressedDiagnosticsCount > 0 {
			response.Mappings[i].SuppressedDiagnostics = make([]uint32, suppressedDiagnosticsCount)
			for j := range suppressedDiagnosticsCount {
				response.Mappings[i].SuppressedDiagnostics[j] = binary.LittleEndian.Uint32(buf[offset:])
				offset += 4
			}
		}
	}

	ignoreMappingsCount := binary.LittleEndian.Uint32(buf[offset:])
	ignoreMappingsByteLen := ignoreMappingsCount * uint32(unsafe.Sizeof(mapping.IgnoreDirectiveMapping{}))
	offset += 4
	response.IgnoreMappings = make([]mapping.IgnoreDirectiveMapping, ignoreMappingsCount)
	copy(response.IgnoreMappings, unsafe.Slice((*mapping.IgnoreDirectiveMapping)(unsafe.Pointer(unsafe.SliceData(buf[offset:offset+ignoreMappingsByteLen]))), ignoreMappingsCount))
	offset += ignoreMappingsByteLen

	expectErrorMappingsCount := binary.LittleEndian.Uint32(buf[offset:])
	expectErrorMappingsByteLen := expectErrorMappingsCount * uint32(unsafe.Sizeof(mapping.ExpectErrorDirectiveMapping{}))
	offset += 4
	response.ExpectErrorMappings = make([]mapping.ExpectErrorDirectiveMapping, expectErrorMappingsCount)
	copy(response.ExpectErrorMappings, unsafe.Slice((*mapping.ExpectErrorDirectiveMapping)(unsafe.Pointer(unsafe.SliceData(buf[offset:offset+expectErrorMappingsByteLen]))), expectErrorMappingsCount))
	offset += expectErrorMappingsByteLen

	return
}
