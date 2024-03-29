package gopylon

/*
#include "camera.h"
*/
import "C"
import (
	"fmt"
	"unsafe"
)

type grabber struct {
	StreamNums   int //支持的stram数量
	ghandle      C.PYLON_STREAMGRABBER_HANDLE
	buffers      [][]uint8
	NumBuffers   int
	bufferHandle []C.PYLON_STREAMBUFFER_HANDLE

	payloadSize int32

	WaitObject WaitObject
}

func (camera Camera) NewGrabber() (*grabber, error) {
	g := &grabber{}
	var ghandle C.PYLON_STREAMGRABBER_HANDLE

	n, err := camera.PylonDeviceGetNumStreamGrabberChannels()
	if err != nil {
		return g, err
	}
	err = camera.PylonDeviceGetStreamGrabber(0, &ghandle)
	if err != nil {
		return g, err
	}

	g.StreamNums = n
	g.ghandle = ghandle
	g.NumBuffers = 5
	for i := 0; i < g.NumBuffers; i++ {
		buf := make([]uint8, g.payloadSize)
		g.buffers = append(g.buffers, buf)

	}
	g.bufferHandle = make([]C.PYLON_STREAMBUFFER_HANDLE, g.NumBuffers)
	return g, err
}

func (g grabber) IsSupportStream() bool {

	return g.StreamNums >= 1
}

func (g grabber) Open() error {
	res := C.CPylonStreamGrabberOpen(g.ghandle)
	if C.GENAPI_E_OK != res {
		return fmt.Errorf("%d", res)
	}

	return nil
}

func (g *grabber) NewWaitObj() (WaitObject, error) {
	wObject := WaitObject{}
	res := C.CPylonStreamGrabberGetWaitObject(g.ghandle, &wObject.Hwait)
	if C.GENAPI_E_OK != res {
		return wObject, fmt.Errorf("%d", res)
	}
	g.WaitObject = wObject
	return wObject, nil
}

func (g grabber) StopClose() error {
	res := C.CPylonStreamGrabberFinishGrab(g.ghandle)
	if C.GENAPI_E_OK != res {
		return fmt.Errorf("%d", res)
	}
	res = C.CPylonStreamGrabberClose(g.ghandle)
	if C.GENAPI_E_OK != res {
		return fmt.Errorf("%d", res)
	}
	return nil
}

func (g grabber) SetMaxNumBuffer() error {
	res := C.CPylonStreamGrabberSetMaxNumBuffer(g.ghandle, C.ulong(g.NumBuffers))
	if C.GENAPI_E_OK != res {
		return fmt.Errorf("%d", res)
	}
	return nil
}

func (g grabber) SetMaxBufferSize() error {
	res := C.CPylonStreamGrabberSetMaxBufferSize(g.ghandle, C.ulong(g.payloadSize))
	if C.GENAPI_E_OK != res {
		return fmt.Errorf("%d", res)
	}
	return nil
}

func (g grabber) PrepareGrab() error {
	res := C.CPylonStreamGrabberPrepareGrab(g.ghandle)
	if C.GENAPI_E_OK != res {
		C.CPrintError(res)
		return fmt.Errorf("%s, %#08x", res)
	}
	return nil
}

func (g grabber) RegisterBuffer() error {

	for i := 0; i < g.NumBuffers; i++ {
		res := C.CPylonStreamGrabberRegisterBuffer(g.ghandle, unsafe.Pointer(&g.buffers[i]), C.ulong(g.payloadSize), &g.bufferHandle[i])
		if C.GENAPI_E_OK != res {
			return fmt.Errorf("%d", res)
		}
	}
	return nil
}

func (g grabber) DeRegisterBuffer() error {
	fmt.Println("de register buffer")
	for i := 0; i < g.NumBuffers; i++ {
		res := C.CPylonStreamGrabberDeregisterBuffer(g.ghandle, g.bufferHandle[i])
		if C.GENAPI_E_OK != res {
			return fmt.Errorf("%d", res)
		}
	}
	return nil
}

/*
* Feed the buffers into the stream grabber's input queue. For each buffer, the API
* allows passing in a pointer to additional context information. This pointer
* will be returned unchanged when the grab is finished. In our example, we use the index of the
* buffer as context information.
 */
func (g grabber) QueueBuffer() error {
	for i := 0; i < g.NumBuffers; i++ {
		res := C.CPylonStreamGrabberQueueBuffer(g.ghandle, g.bufferHandle[i], unsafe.Pointer(&i))
		if C.GENAPI_E_OK != res {
			C.CPrintError(res)
			return fmt.Errorf("%d", res)
		}
	}

	return nil
}

func (g *grabber) GetPayloadSize(hdev C.PYLON_DEVICE_HANDLE) error {
	var value C.size_t
	res := C.CPylonStreamGrabberGetPayloadSize(hdev, g.ghandle, &value)
	if C.GENAPI_E_OK != res {
		return fmt.Errorf("%d", res)
	}
	g.payloadSize = int32(value)
	return nil
}

func (g grabber) StartStreamingIfMandatory() error {

	/* Start the image acquisition engine. */
	res := C.CPylonStreamGrabberStartStreamingIfMandatory(g.ghandle)
	if C.GENAPI_E_OK != res {
		return fmt.Errorf("%d", res)
	}
	return nil
}

func (g grabber) StopStreamingIfMandatory() error {

	/* Start the image acquisition engine. */
	res := C.CPylonStreamGrabberStopStreamingIfMandatory(g.ghandle)
	if C.GENAPI_E_OK != res {
		return fmt.Errorf("%d", res)
	}
	return nil
}

func (g grabber) RetrieveResult() (C.PylonGrabResult_t, error) {
	var rslt C.PylonGrabResult_t
	var isReady C._Bool
	res := C.CPylonStreamGrabberRetrieveResult(g.ghandle, &rslt, &isReady)
	if C.GENAPI_E_OK != res {
		return rslt, fmt.Errorf("%d", res)
	}
	if !isReady {
		return rslt, fmt.Errorf("retrieve failed")
	}

	bufferIndex := *(*uint32)(unsafe.Pointer(rslt.Context))

	fmt.Println("bufferIndex:", bufferIndex)

	switch rslt.Status {
	case C.Idle:
	case C.Queued:
	case C.Grabbed:
		fmt.Printf("Grabbed frame into buffer %2d. sizeX: %d. sizeY: %d \n", bufferIndex, rslt.SizeX, rslt.SizeY)
		//save image
	case C.Canceled:
	case C.Failed:

	default:
	}

	res = C.CPylonStreamGrabberQueueBuffer(g.ghandle, rslt.hBuffer, unsafe.Pointer(&bufferIndex))
	if C.GENAPI_E_OK != res {
		C.CPrintError(res)
		return rslt, fmt.Errorf("%d", res)
	}
	return rslt, nil
}

func (g grabber) FlushBuffersToOutput() error {
	res := C.CPylonStreamGrabberFlushBuffersToOutput(g.ghandle)
	if C.GENAPI_E_OK != res {
		C.CPrintError(res)
		return fmt.Errorf("%d", res)
	}
	return nil
}

func (g grabber) JustRetrieveResult() (C.PylonGrabResult_t, error, bool) {
	var rslt C.PylonGrabResult_t
	var isReady C._Bool
	res := C.CPylonStreamGrabberRetrieveResult(g.ghandle, &rslt, &isReady)
	if C.GENAPI_E_OK != res {
		C.CPrintError(res)
		return rslt, fmt.Errorf("%d", res), bool(isReady)
	}
	return rslt, nil, bool(isReady)
}
