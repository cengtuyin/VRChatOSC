package screen

import (
	"fmt"
	"path/filepath"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	EVENT_SYSTEM_FOREGROUND = 0x0003
	WINEVENT_OUTOFCONTEXT   = 0x0000
)

var (
	user32                       = windows.NewLazySystemDLL("user32.dll")
	kernel32                     = windows.NewLazySystemDLL("kernel32.dll")
	procSetWinEventHook          = user32.NewProc("SetWinEventHook")
	procUnhookWinEvent           = user32.NewProc("UnhookWinEvent")
	procGetWindowTextW           = user32.NewProc("GetWindowTextW")
	procGetWindowTextLengthW     = user32.NewProc("GetWindowTextLengthW")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procGetModuleFileNameW       = kernel32.NewProc("GetModuleFileNameW")
	procOpenProcess              = kernel32.NewProc("OpenProcess")
	procCloseHandle              = kernel32.NewProc("CloseHandle")
	procGetMessage               = user32.NewProc("GetMessageW")
	procTranslateMessage         = user32.NewProc("TranslateMessage")
	procDispatchMessage          = user32.NewProc("DispatchMessageW")
)

// AppInfo 应用信息结构体
type AppInfo struct {
	WindowTitle string  // 窗口标题
	ProcessName string  // 进程名（如 notepad.exe）
	ProcessPath string  // 进程完整路径
	HWND        uintptr // 窗口句柄
}

var hook syscall.Handle

// 获取窗口标题
func getWindowTitle(hwnd uintptr) string {
	length, _, _ := procGetWindowTextLengthW.Call(hwnd)
	if length == 0 {
		return ""
	}
	buf := make([]uint16, length+1)
	procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(length+1))
	return syscall.UTF16ToString(buf)
}

// 获取进程完整路径和名称
func getProcessInfo(hwnd uintptr) (processPath, processName string) {
	var pid uint32
	procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if pid == 0 {
		return "", ""
	}

	handle, _, _ := procOpenProcess.Call(
		windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_VM_READ,
		0,
		uintptr(pid),
	)
	if handle == 0 {
		return "", ""
	}
	defer procCloseHandle.Call(handle)

	buf := make([]uint16, windows.MAX_PATH)
	ret, _, _ := procGetModuleFileNameW.Call(handle, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if ret == 0 {
		return "", ""
	}

	processPath = syscall.UTF16ToString(buf)
	processName = filepath.Base(processPath)
	return processPath, processName
}

// 获取当前焦点应用信息
func GetCurrentForegroundApp() AppInfo {
	hwnd, _, _ := user32.NewProc("GetForegroundWindow").Call()
	if hwnd == 0 {
		return AppInfo{}
	}

	title := getWindowTitle(hwnd)
	processPath, processName := getProcessInfo(hwnd)

	return AppInfo{
		WindowTitle: title,
		ProcessName: processName,
		ProcessPath: processPath,
		HWND:        hwnd,
	}
}

// 事件回调
func winEventProc(_ syscall.Handle, _ uint32, hwnd uintptr, _ int32, _ int32, _ uint32, _ uint32) {
	// if event == EVENT_SYSTEM_FOREGROUND && hwnd != 0 {
	title := getWindowTitle(hwnd)
	processPath, processName := getProcessInfo(hwnd)

	fmt.Printf("[焦点变化]\n")
	fmt.Printf("  窗口句柄: %d\n", hwnd)
	fmt.Printf("  窗口标题: %s\n", title)
	fmt.Printf("  应用名称: %s\n", processName)
	fmt.Printf("  应用路径: %s\n", processPath)
	fmt.Println("---")
	// }
}

// ListenTopapp 启动焦点监听（阻塞式）
func ListenTopapp() {
	callback := syscall.NewCallback(func(hook syscall.Handle, event uint32, hwnd uintptr, idObject int32, idChild int32, dwEventThread uint32, dwmsEventTime uint32) uintptr {
		winEventProc(hook, event, hwnd, idObject, idChild, dwEventThread, dwmsEventTime)
		return 0
	})

	ret, _, err := procSetWinEventHook.Call(
		EVENT_SYSTEM_FOREGROUND,
		EVENT_SYSTEM_FOREGROUND,
		0,
		callback,
		0, 0,
		WINEVENT_OUTOFCONTEXT,
	)
	if ret == 0 {
		fmt.Printf("安装钩子失败: %v\n", err)
		return
	}
	hook = syscall.Handle(ret)
	defer procUnhookWinEvent.Call(uintptr(hook))

	fmt.Println("开始监听焦点应用变化...")
	fmt.Println("提示：切换窗口即可看到应用名称")

	// 消息循环
	var msg struct {
		HWND    uintptr
		Message uint32
		WParam  uintptr
		LParam  uintptr
		Time    uint32
		Pt      struct{ X, Y int32 }
	}

	for {
		ret, _, _ := procGetMessage.Call(
			uintptr(unsafe.Pointer(&msg)),
			0,
			0,
			0,
		)
		if ret == 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessage.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

// GetCurrentAppName 简单的获取当前应用名称的函数
func GetCurrentAppName() string {
	info := GetCurrentForegroundApp()
	if info.ProcessName != "" {
		return info.ProcessName
	}
	return info.WindowTitle
}
