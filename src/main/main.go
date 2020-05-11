// +build windows

package main

import (
	"log"
	"os"
	"os/signal"
	"strconv"
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
var logE = log.New(os.Stderr, "ERROR: ", log.LstdFlags)
var logI = log.New(os.Stdout, "INFO: ", log.LstdFlags)
var configScaleYLin int32
var configScaleYQuad int32

var defaultConfig = map[string]string{
	"TRACK_SCALE_Y_LINEAR":    "1",
	"TRACK_SCALE_Y_QUADRATIC": "1",
}

func main() {
	if err := run(); err != nil {
		logE.Fatal(err)
	}
}

// Use a value set in an environment variable, else use default
func config(key string) string {
	val := os.Getenv(key)
	if val == "" {
		val = defaultConfig[key]
	}
	logI.Printf("Using %s=%s", key, val)
	return val
}

func run() (err error) {

	logI.Printf("Checking environment variables for configuration")

	configScaleYLin_64, err := strconv.ParseInt(config("TRACK_SCALE_Y_LINEAR"), 0, 32)
	if err != nil {
		return err
	}
	configScaleYLin = int32(configScaleYLin_64)

	configScaleYQuad_64, err := strconv.ParseInt(config("TRACK_SCALE_Y_QUADRATIC"), 0, 32)
	if err != nil {
		return err
	}
	configScaleYQuad = int32(configScaleYQuad_64)

	// Not sure what the buffer size should be?
	mouseChan := make(chan types.MouseEvent, 100)
	if err := mouse.Install(mouseHandler, mouseChan); err != nil {
		return err
	}
	defer mouse.Uninstall()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)

	logI.Println("Start capturing mouse input")

	for {
		select {
		case <-signalChan:
			logI.Println("Received shutdown signal")
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
				yDiff := (mY - m.Y)
				moveAmount := yDiff * configScaleYLin
				if configScaleYQuad != 0 {
					if yDiff > 0 {
						moveAmount += yDiff * yDiff * configScaleYQuad / 64
					} else {
						moveAmount -= yDiff * yDiff * configScaleYQuad / 64
					}
				}
				go procMouseEvent.Call(moveWheel, 0, 0, uintptr(moveAmount), 0)
				return 1
			}
		}

	NEXT:
		return win32.CallNextHookEx(0, code, wParam, lParam)
	}
}
