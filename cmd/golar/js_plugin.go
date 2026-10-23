package main

/*
#include <stdint.h>
#include <stdlib.h>

void napi_call_threadsafe_function_any(uintptr_t func, uintptr_t data, size_t is_blocking);
*/
import "C"

import (
	"encoding/binary"
	"encoding/json"
	"sync"

	"github.com/auvred/golar/internal/golar"
	"github.com/auvred/golar/internal/tscodegenplugin"
	"github.com/auvred/golar/plugin"
)

var jsPluginsMu sync.Mutex
var jsPluginHost *tscodegenplugin.JsPluginHost
var jsPlugins []*tscodegenplugin.JsPlugin

type ipcCodegenPluginRegistration struct {
	Cmd        []string               `json:"cmd"`
	Extensions []plugin.FileExtension `json:"extensions"`
}

//export golar_js_registerJsCodegen
func golar_js_registerJsCodegen(threadId C.uint32_t, createServiceCodeTsfn C.uintptr_t) {
	buf := syncBuffers[threadId]

	var offset uint32
	pluginId := binary.LittleEndian.Uint32(buf[offset:])
	offset += 4

	jsPluginsMu.Lock()
	defer jsPluginsMu.Unlock()

	if jsPluginHost == nil {
		jsPluginHost = tscodegenplugin.NewJsPluginHost()
	}

	if uint32(len(jsPlugins)) < pluginId+1 {
		jsPlugins = append(jsPlugins, make([]*tscodegenplugin.JsPlugin, pluginId+1-uint32(len(jsPlugins)))...)
	}

	if jsPlugins[pluginId] == nil {
		jsPlugins[pluginId] = jsPluginHost.NewJsPlugin(buf[offset:])
		golar.RegisterCodegenPlugin(jsPlugins[pluginId])
	}

	jsPluginHost.EnsureWorkerSpawned(uint32(threadId), buf)
	jsPlugins[pluginId].RegisterWorkerCallback(uint32(threadId), func() {
		C.napi_call_threadsafe_function_any(createServiceCodeTsfn, 0, 0)
	})
}

//export golar_js_jsCodegenCreateServiceCodeResponse
func golar_js_jsCodegenCreateServiceCodeResponse(threadId C.uint32_t) {
	buf := syncBuffers[threadId]
	tscodegenplugin.JsPluginHandleCreateServiceCodeResponse(buf)
}

//export golar_js_jsCodegenCreateServiceCodeFailed
func golar_js_jsCodegenCreateServiceCodeFailed(threadId C.uint32_t, messageData *C.char, messageLen C.size_t) {
	jsPluginsMu.Lock()
	host := jsPluginHost
	jsPluginsMu.Unlock()
	host.HandleCreateServiceCodeFailure(uint32(threadId), C.GoStringN(messageData, C.int(messageLen)))
}

//export golar_js_registerIpcCodegen
func golar_js_registerIpcCodegen() {
	buf := syncBuffers[0]
	payloadLen := binary.LittleEndian.Uint32(buf)
	registration := ipcCodegenPluginRegistration{}
	if err := json.Unmarshal(buf[4:4+payloadLen], &registration); err != nil {
		panic(err)
	}

	plugin, err := tscodegenplugin.NewIpcPlugin(registration.Cmd, registration.Extensions)
	if err != nil {
		panic(err)
	}
	golar.RegisterCodegenPlugin(plugin)
}
