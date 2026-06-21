package keyboard

import (
	"fmt"
	"os/exec"
	"syscall"
	"time"
	"unsafe"
	"vrchat_osc/module/config"
	"vrchat_osc/module/iot"

	"github.com/hypebeast/go-osc/osc"
)

// Windows API 常量
const (
	WH_KEYBOARD_LL = 13
	WM_KEYDOWN     = 0x0100
)

// 回调函数类型
type HookProc func(code int, wParam uintptr, lParam uintptr) uintptr

// 键盘事件结构体（低层键盘钩子专用）
type KBDLLHOOKSTRUCT struct {
	VkCode      uint32
	ScanCode    uint32
	Flags       uint32
	Time        uint32
	DwExtraInfo uintptr
}

var (
	user32           = syscall.NewLazyDLL("user32.dll")
	setWindowsHookEx = user32.NewProc("SetWindowsHookExW")
	callNextHookEx   = user32.NewProc("CallNextHookEx")
	getMessage       = user32.NewProc("GetMessageW")
	translateMessage = user32.NewProc("TranslateMessage")
	dispatchMessage  = user32.NewProc("DispatchMessageW")
)

// 全局钩子句柄
var hookHandle uintptr

// 回调函数（必须作为全局变量保留引用，避免被 GC）
var keyboardCallback HookProc

// 导出变量（首字母大写）
var (
	KeyboardInputSecond int64 // 上次发送 typing 的时间
)

// 键盘钩子回调（必须使用 syscall 规定的调用约定）
func keyboardProc(code int, wParam uintptr, lParam uintptr) uintptr {
	if code >= 0 && wParam == WM_KEYDOWN {
		if iot.OffComputerTimeout {
			iot.OffComputerTimeout = false
			exec.Command("cmd", "/c", "shutdown", "/a").Run()
			client := osc.NewClient("localhost", config.VRChatOSCPort)
			msg := osc.NewMessage("/chatbox/input")
			msg.Append("- 远程关机被打断 -")
			msg.Append(true)
			msg.Append(false)
			client.Send(msg)
		}

		// 解析键盘事件结构体
		kbd := (*KBDLLHOOKSTRUCT)(unsafe.Pointer(lParam))
		config.KeyboardInputTime = time.Now().Unix()
		// println(kbd.VkCode)
		if time.Now().Unix()-KeyboardInputSecond < 15 {
			KeyboardInputSecond = time.Now().Unix()
		}
		if kbd.VkCode == 89 && config.TopApplicationTitle == "VRChat" { // Y 键
			if time.Now().Unix()-KeyboardInputSecond >= 60 {
				config.ContinueNextMessage = true
				KeyboardInputSecond = time.Now().Unix()
				client := osc.NewClient("localhost", config.VRChatOSCPort)
				msg := osc.NewMessage("/chatbox/input")
				msg.Append("typing")
				msg.Append(true)
				msg.Append(false)
				client.Send(msg)
			}
		}
	}
	// 传递给下一个钩子（保证系统正常行为）
	ret, _, _ := callNextHookEx.Call(0, uintptr(code), wParam, lParam)
	return ret
}

// KeyboardInput 启动键盘监听（首字母大写，公开方法）
func KeyboardInput() {
	// 设置回调函数
	keyboardCallback = func(code int, wParam, lParam uintptr) uintptr {
		return keyboardProc(code, wParam, lParam)
	}

	// 安装键盘钩子
	hook, _, err := setWindowsHookEx.Call(
		WH_KEYBOARD_LL,
		syscall.NewCallback(keyboardCallback),
		uintptr(0), // 低层钩子不需要模块句柄
		0,          // 全局钩子线程 ID 为 0
	)
	if hook == 0 {
		fmt.Println("安装钩子失败:", err)
		return
	}
	hookHandle = hook

	// 消息循环（钩子需要消息泵来触发回调）
	var msg struct {
		HWND    uintptr
		Message uint32
		WParam  uintptr
		LParam  uintptr
		Time    uint32
		Pt      struct{ X, Y int32 }
	}
	for {
		ret, _, _ := getMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if ret == 0 { // WM_QUIT
			break
		}
		translateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		dispatchMessage.Call(uintptr(unsafe.Pointer(&msg)))
	}
}
