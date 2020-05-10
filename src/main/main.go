// +build windows

package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"unsafe"

	"github.com/moutend/go-hook/pkg/mouse"
	"github.com/moutend/go-hook/pkg/types"
	"github.com/moutend/go-hook/pkg/win32"
)

var dll = syscall.NewLazyDLL("user32.dll")

// See: https://docs.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-mouse_event
var procMouseEvent = dll.NewProc("mouse_event")
var moveWheel = uintptr(0x0800)

func main() {
	log.SetFlags(0)
	log.SetPrefix("error: ")

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() (err error) {
	// Not sure what the buffer size should be?
	mouseChan := make(chan types.MouseEvent, 100)
	if err := mouse.Install(mouseHandler, mouseChan); err != nil {
		return err
	}
	defer mouse.Uninstall()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)

	fmt.Println("Start capturing mouse input")

	for {
		select {
		case <-signalChan:
			fmt.Println("Received shutdown signal")
			return nil
		case <-mouseChan:
			continue
		}
	}
}

func mouseHandler(c chan<- types.MouseEvent) types.HOOKPROC {

	var mY int32
	b1 := false
	b2 := false
	state := 0

	return func(code int32, wParam, lParam uintptr) uintptr {

		if code < 0 {
			goto NEXT
		}
		if lParam == 0 {
			goto NEXT
		}

		c <- types.MouseEvent{
			Message:        types.Message(wParam),
			MSLLHOOKSTRUCT: *(*types.MSLLHOOKSTRUCT)(unsafe.Pointer(lParam)),
		}

		switch types.Message(wParam) {

		case types.WM_LBUTTONDOWN:
			b1 = true
			if b1 && b2 {
				state = 1
			}
		case types.WM_LBUTTONUP:
			b1 = false
			state = 0
		case types.WM_RBUTTONDOWN:
			b2 = true
			if b1 && b2 {
				state = 1
			}
		case types.WM_RBUTTONUP:
			b2 = false
			state = 0
		case types.WM_MOUSEMOVE:
			m := (*types.MSLLHOOKSTRUCT)(unsafe.Pointer(lParam))
			switch state {
			case 1:
				state = 2
				mY = m.Y
			case 2:
				moveAmount := (mY - m.Y)
				if moveAmount > 0 {
					moveAmount *= moveAmount
				} else {
					moveAmount *= -moveAmount
				}
				go procMouseEvent.Call(moveWheel, 0, 0, uintptr(moveAmount), 0)
				return 1
			}
		}

	NEXT:
		return win32.CallNextHookEx(0, code, wParam, lParam)
	}
}
