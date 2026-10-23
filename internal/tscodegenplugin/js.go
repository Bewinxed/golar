package tscodegenplugin

import (
	"encoding/binary"
	"encoding/json"
	"sync"
	"time"
	"unsafe"

	"github.com/auvred/golar/plugin"
)

type jsPluginRegistration struct {
	Extensions []plugin.FileExtension `json:"extensions"`
}

type JsPluginHost struct {
	workers [64]*JsPluginWorker
}

func NewJsPluginHost() *JsPluginHost {
	return &JsPluginHost{}
}

func (p *JsPluginHost) EnsureWorkerSpawned(workerId uint32, buf []byte) {
	if p.workers[workerId] != nil {
		return
	}
	w := &JsPluginWorker{
		id:    workerId,
		buf:   buf,
		queue: make(chan jsCreateServiceCodeRequest, 100),
	}
	p.workers[workerId] = w
	go w.spawn()
}

func (h *JsPluginHost) NewJsPlugin(buf []byte) *JsPlugin {
	var offset uint32
	initializationLen := binary.LittleEndian.Uint32(buf[offset:])
	offset += 4
	var registration jsPluginRegistration
	if err := json.Unmarshal(buf[offset:offset+initializationLen], &registration); err != nil {
		panic(err)
	}
	return &JsPlugin{
		host:       h,
		queue:      make(chan jsCreateServiceCodeRequest, 100),
		extensions: registration.Extensions,
	}
}

type jsCreateServiceCodeRequest struct {
	req                   CreateServiceCodeRequest
	callCreateServiceCode func(workerId uint32)
	resp                  chan<- CreateServiceCodeResponse
}

type jsCreateServiceCodePendingRequest struct {
	started  time.Time
	fileName string
	callback func(response CreateServiceCodeResponse)
}

type JsPluginWorker struct {
	id             uint32
	buf            []byte
	queue          chan jsCreateServiceCodeRequest
	pendingRequest jsCreateServiceCodePendingRequest
}

func (w *JsPluginWorker) spawn() {
	var wg sync.WaitGroup
	for req := range w.queue {
		var response CreateServiceCodeResponse

		wg.Add(1)

		w.pendingRequest = jsCreateServiceCodePendingRequest{
			started:  time.Now(),
			fileName: req.req.FileName,
			callback: func(r CreateServiceCodeResponse) {
				defer wg.Done()
				response = r
			},
		}

		offset := 0
		binary.LittleEndian.PutUint64(w.buf[offset:], uint64(uintptr(unsafe.Pointer(w))))
		offset += 8
		encodeCreateServiceCodeRequest(w.buf[offset:], req.req)

		req.callCreateServiceCode(w.id)

		wg.Wait()
		req.resp <- response
	}
}

func JsPluginHandleCreateServiceCodeResponse(buf []byte) {
	offset := 0
	w := (*JsPluginWorker)(unsafe.Pointer(uintptr(binary.LittleEndian.Uint64(buf[offset:]))))
	offset += 8
	w.pendingRequest.callback(decodeCreateServiceCodeResponse(w.buf[offset:]))
}

// HandleCreateServiceCodeFailure answers the request pending on workerId's
// thread, whose createServiceCode threw or was rejected, with message as the
// file's service code error.
func (h *JsPluginHost) HandleCreateServiceCodeFailure(workerId uint32, message string) {
	h.workers[workerId].pendingRequest.callback(CreateServiceCodeResponse{
		Errors: []ServiceCodeError{{Message: message}},
	})
}

type JsPlugin struct {
	host                       *JsPluginHost
	extensions                 []plugin.FileExtension
	queue                      chan jsCreateServiceCodeRequest
	createServiceCodeCallbacks [64]func()
}

var _ Plugin = (*JsPlugin)(nil)

func (p *JsPlugin) RegisterWorkerCallback(workerId uint32, callCreateServiceCode func()) {
	p.createServiceCodeCallbacks[workerId] = callCreateServiceCode
	go func() {
		for req := range p.queue {
			p.host.workers[workerId].queue <- req
		}
	}()
}

func (p *JsPlugin) Extensions() []plugin.FileExtension {
	return p.extensions
}

func (p *JsPlugin) CreateServiceCode(req CreateServiceCodeRequest) CreateServiceCodeResponse {
	resp := make(chan CreateServiceCodeResponse)

	p.queue <- jsCreateServiceCodeRequest{
		req:                   req,
		callCreateServiceCode: p.callCreateServiceCode,
		resp:                  resp,
	}

	res := <-resp
	return res
}

func (p *JsPlugin) callCreateServiceCode(workerId uint32) {
	p.createServiceCodeCallbacks[workerId]()
}
