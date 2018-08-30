package main

import (
	"syscall"
	"golang.org/x/sys/windows"
	"unsafe"
	"fmt"
	"time"
	"os/exec"
	"os"
	"bytes"
	"encoding/gob"
	"io/ioutil"
	"sort"
)

var (
	count map[DWORD]uint64
	filename string
)

var (
	user32                  = windows.NewLazySystemDLL("user32.dll")
	procSetWindowsHookEx    = user32.NewProc("SetWindowsHookExW")
	procCallNextHookEx      = user32.NewProc("CallNextHookEx")
	procUnhookWindowsHookEx = user32.NewProc("UnhookWindowsHookEx")
	procGetMessage          = user32.NewProc("GetMessageW")
	procTranslateMessage    = user32.NewProc("TranslateMessage")
	procDispatchMessage     = user32.NewProc("DispatchMessageW")
	keyboardHook            HHOOK
	mouseHook               HHOOK
)

const (
	WH_KEYBOARD_LL = 13
	WH_MOUSE_LL    = 14
	WH_KEYBOARD    = 2
	WM_KEYDOWN     = 256
	WM_SYSKEYDOWN  = 260
	WM_KEYUP       = 257
	WM_SYSKEYUP    = 261
	WM_KEYFIRST    = 256
	WM_KEYLAST     = 264
	PM_NOREMOVE    = 0x000
	PM_REMOVE      = 0x001
	PM_NOYIELD     = 0x002
	WM_LBUTTONDOWN = 0x0201
	WM_RBUTTONDOWN = 0x0204
	NULL           = 0

	VK_LBUTTON = 0x01
	VK_RBUTTON = 0X02
)

type (
	DWORD uint32
	WPARAM uintptr
	LPARAM uintptr
	LRESULT uintptr
	HANDLE uintptr
	HINSTANCE HANDLE
	HHOOK HANDLE
	HWND HANDLE
)

type HOOKPROC func(int, WPARAM, LPARAM) LRESULT

type KBDLLHOOKSTRUCT struct {
	VkCode      DWORD
	ScanCode    DWORD
	Flags       DWORD
	Time        DWORD
	DwExtraInfo uintptr
}

// http://msdn.microsoft.com/en-us/library/windows/desktop/dd162805.aspx
type POINT struct {
	X, Y int32
}

// http://msdn.microsoft.com/en-us/library/windows/desktop/ms644958.aspx
type MSG struct {
	Hwnd    HWND
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
}

type tagMSLLHOOKSTRUCT struct {
	POINT       POINT
	MouseData   DWORD
	Flags       DWORD
	Time        DWORD
	DwExtraInfo uintptr
}
type MSLLHOOKSTRUCT tagMSLLHOOKSTRUCT
type LPMSLLHOOKSTRUCT tagMSLLHOOKSTRUCT
type PMSLLHOOKSTRUCT tagMSLLHOOKSTRUCT

func SetWindowsHookEx(idHook int, lpfn HOOKPROC, hMod HINSTANCE, dwThreadId DWORD) HHOOK {
	ret, _, _ := procSetWindowsHookEx.Call(
		uintptr(idHook),
		uintptr(syscall.NewCallback(lpfn)),
		uintptr(hMod),
		uintptr(dwThreadId),
	)
	return HHOOK(ret)
}

func ToUnicodeEx() {

}

func CallNextHookEx(hhk HHOOK, nCode int, wParam WPARAM, lParam LPARAM) LRESULT {
	ret, _, _ := procCallNextHookEx.Call(
		uintptr(hhk),
		uintptr(nCode),
		uintptr(wParam),
		uintptr(lParam),
	)
	return LRESULT(ret)
}

func UnhookWindowsHookEx(hhk HHOOK) bool {
	ret, _, _ := procUnhookWindowsHookEx.Call(
		uintptr(hhk),
	)
	return ret != 0
}

func GetMessage(msg *MSG, hwnd HWND, msgFilterMin uint32, msgFilterMax uint32) int {
	ret, _, _ := procGetMessage.Call(
		uintptr(unsafe.Pointer(msg)),
		uintptr(hwnd),
		uintptr(msgFilterMin),
		uintptr(msgFilterMax))
	return int(ret)
}

func TranslateMessage(msg *MSG) bool {
	ret, _, _ := procTranslateMessage.Call(
		uintptr(unsafe.Pointer(msg)))
	return ret != 0
}

func DispatchMessage(msg *MSG) uintptr {
	ret, _, _ := procDispatchMessage.Call(
		uintptr(unsafe.Pointer(msg)))
	return ret
}

func Start() {
	// defer user32.Release()
	keyboardHook = SetWindowsHookEx(WH_KEYBOARD_LL,
		(HOOKPROC)(func(nCode int, wparam WPARAM, lparam LPARAM) LRESULT {
			if nCode == 0 && wparam == WM_KEYDOWN {
				kbdstruct := (*KBDLLHOOKSTRUCT)(unsafe.Pointer(lparam))
				//fmt.Printf("%#v\n", kbdstruct)
				//code := rune(kbdstruct.VkCode)
				//fmt.Printf("%q\n", code)
				count[kbdstruct.VkCode]++
			}
			return CallNextHookEx(keyboardHook, nCode, wparam, lparam)
		}), 0, 0)
	mouseHook = SetWindowsHookEx(WH_MOUSE_LL,
		(HOOKPROC)(func(nCode int, wparam WPARAM, lparam LPARAM) LRESULT {
			if nCode == 0 && (wparam == WM_KEYDOWN || wparam == WM_LBUTTONDOWN || wparam == WM_RBUTTONDOWN) {
				//msstruct := (*MSLLHOOKSTRUCT)(unsafe.Pointer(lparam))
				//fmt.Printf("%#v\n", msstruct)
				if wparam == WM_LBUTTONDOWN {
					//fmt.Println("left")
					count[VK_LBUTTON]++
				} else if wparam == WM_RBUTTONDOWN {
					//fmt.Println("right")
					count[VK_RBUTTON]++
				} else {
					//fmt.Println("none")
				}
			}
			return CallNextHookEx(keyboardHook, nCode, wparam, lparam)
		}), 0, 0)

	var msg MSG
	for GetMessage(&msg, 0, 0, 0) != 0 {
		// TranslateMessage(msg)
		// DispatchMessage(msg)
	}

	UnhookWindowsHookEx(keyboardHook)
	UnhookWindowsHookEx(mouseHook)
	keyboardHook = 0
	mouseHook = 0
}

func Print() {
	i := 0
	c := exec.Command("cmd", "/c", "cls")
	c.Stdout = os.Stdout
	c.Run()
	keys := make([]int, len(count))
	j := 0
	for k := range count {
		keys[j] = int(k)
		j++
	}
	sort.Ints(keys)
	for _, k := range keys {
		vkcode := DWORD(k)
		num := count[vkcode]
		code := rune(vkcode)
		fmt.Printf("%x(%q):%d\t", vkcode, code, num)
		i++
		if i%5 == 0 {
			fmt.Println()
		}
	}
}

func Save() {
	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	err := enc.Encode(count)
	if err != nil {
		panic(err)
	}
	name := fmt.Sprintf("%d%02d%02d", time.Now().Year(), time.Now().Month(), time.Now().Day())
	if name!=filename{
		count=make(map[DWORD]uint64)
		filename=name
	}
	err = ioutil.WriteFile(filename, buff.Bytes(), 0644)
	if err != nil {
		panic(nil)
	}
}

func Load() {
	filename = fmt.Sprintf("%d%02d%02d", time.Now().Year(), time.Now().Month(), time.Now().Day())
	fmt.Println(filename)
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}
	buffer := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buffer)
	err = dec.Decode(&count)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%#v", count)
}

func main() {
	count = make(map[DWORD]uint64)
	Load()
	go func() {
		for {
			Print()
			Save()
			time.Sleep(5 * time.Second)
		}
	}()
	Start()
}
