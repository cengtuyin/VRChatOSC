package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
	"vrchat_osc/module/adb"
	"vrchat_osc/module/config"
	"vrchat_osc/module/iot"
	"vrchat_osc/module/keyboard"
	"vrchat_osc/module/music"
	"vrchat_osc/module/screen"

	"github.com/hypebeast/go-osc/osc"
)

var (
	last_desktop_app string
	pet_status       bool

	hideRules                 []Rule
	titleRules                []Rule
	hidetextRules             []Rule
	user_config               map[string]any
	usenetworkdevice          bool   = false
	typingtimeout             int64  = 60
	afk                       bool   = true
	afk_timeout               int64  = 900
	afk_clock                 int64  = 30
	autooffcomputer           bool   = false
	autooffcomputertimeout    int64  = 7200
	applicationheadtext       string = "[💻]USING: "
	music_showartist          bool   = true
	music_headtext            string = "MUSIC: "
	music_delimiter           string = "@"
	phone                     bool   = false
	phone_applicationheadtext string = "[📱]USING: "
	batteryheadtext           string = "BATTERY: "
	batteryheadtextpower      string = "BATTERY: ⚡️"
)

// Rule 单条匹配规则
type Rule struct {
	Name string            `json:"name"`
	From string            `json:"from"`
	To   string            `json:"to,omitempty"`
	Rule map[string]string `json:"rule,omitempty"`
}

func disableQuickEditMode() {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getConsoleMode := kernel32.NewProc("GetConsoleMode")
	setConsoleMode := kernel32.NewProc("SetConsoleMode")

	var mode uint32
	// 获取标准输入句柄（通常是 0，即 STD_INPUT_HANDLE）
	handle, _, _ := getConsoleMode.Call(uintptr(syscall.Stdin))

	// ENABLE_QUICK_EDIT_MODE = 0x0040
	const ENABLE_QUICK_EDIT_MODE = 0x0040

	getConsoleMode.Call(uintptr(syscall.Stdin), uintptr(unsafe.Pointer(&mode)))
	// 清除快速编辑模式标志
	mode &^= ENABLE_QUICK_EDIT_MODE
	setConsoleMode.Call(uintptr(syscall.Stdin), uintptr(mode))
	_, _, _ = handle, mode, kernel32 // 避免未使用警告
}
func keepProcessAlive() {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	proc := kernel32.NewProc("SetThreadExecutionState")

	// 检查 API 是否存在
	if proc.Find() != nil {
		log.Println("[保活机制] SetThreadExecutionState 不可用，跳过")
		return
	}

	const ES_CONTINUOUS = 0x80000000
	const ES_SYSTEM_REQUIRED = 0x00000001

	go func() {
		for {
			proc.Call(ES_CONTINUOUS | ES_SYSTEM_REQUIRED)
			time.Sleep(20 * time.Second)
		}
	}()
}

func getExeDir() string {
	exePath, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exePath)
}

// CheckCommandAvailable 检查命令是否在 PATH 环境变量中可用
func CheckCommandAvailable(cmd string) (string, bool) {
	// 获取 PATH 环境变量
	pathEnv := os.Getenv("PATH")
	paths := strings.Split(pathEnv, string(os.PathListSeparator))

	// 遍历路径检查命令
	for _, path := range paths {
		fullPath := filepath.Join(path, cmd)
		if _, err := os.Stat(fullPath); err == nil {
			return fullPath, true
		}
	}
	return "", false
}

func main() {
	keepProcessAlive()
	disableQuickEditMode()

	exeDir := getExeDir()
	if err := os.Chdir(exeDir); err != nil {
		log.Printf("切换工作目录失败: %v\n", err)
		return
	}

	os.Remove("vrchat_osc.log")
	logFile, err := os.OpenFile("vrchat_osc.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		// 同时输出到终端和文件
		multiWriter := io.MultiWriter(os.Stdout, logFile)
		log.SetOutput(multiWriter)
	}

	log.Println("=============================")
	log.Println("Made by Rexxrt!")
	log.Println("Public Version.")

	// 加载规则文件
	hideRules = loadRules("./data/rules_hide.json")
	titleRules = loadRules("./data/rules_title.json")
	hidetextRules = loadRules("./data/rules_hidetext.json")
	data, err := os.ReadFile("./data/config.json")
	if err != nil {
		log.Printf("无法读取配置文件(data/config.json)\n")
		return
	}
	if err := json.Unmarshal(data, &user_config); err != nil {
		log.Printf("配置文件错误(data/config.json): %v\n", err)
		return
	}
	if safk, ok := user_config["typingtimeout"]; ok {
		typingtimeout = int64(safk.(float64))
	}
	if safk, ok := user_config["usenetworkdevice"]; ok && safk.(bool) {
		usenetworkdevice = true
		log.Println("自用模式")
	}

	safk, ok := user_config["AFK"].(map[string]any)["status"]
	afk = ok && safk.(bool)
	safk, ok = user_config["autooffcomputer"]
	autooffcomputer = ok && safk.(bool)
	if safk, ok = user_config["autooffcomputertimeout"]; ok {
		autooffcomputertimeout = int64(safk.(float64))
	}
	if safk, ok = user_config["AFK"].(map[string]any)["timeout"]; ok {
		afk_timeout = int64(safk.(float64))
	}
	if safk, ok = user_config["clock"]; ok {
		config.WaitTime = int64(safk.(float64))
	}
	if safk, ok = user_config["AFK"].(map[string]any)["clock"]; ok {
		afk_clock = int64(safk.(float64))
	}
	if safk, ok = user_config["applicationheadtext"]; ok {
		applicationheadtext = safk.(string)
	}
	if safk, ok = user_config["music"].(map[string]any)["headtext"]; ok {
		music_headtext = safk.(string)
	}
	if safk, ok = user_config["music"].(map[string]any)["music_delimiter"]; ok {
		music_delimiter = safk.(string)
	}
	safk, ok = user_config["phone"].(map[string]any)["status"]
	phone = ok && safk.(bool)
	if _, st := CheckCommandAvailable("adb.exe"); !st {
		phone = false
		log.Println("未找到 adb.exe，手机状态功能已关闭")
	}
	if safk, ok = user_config["phone"].(map[string]any)["applicationheadtext"]; ok {
		phone_applicationheadtext = safk.(string)
	}
	if safk, ok = user_config["phone"].(map[string]any)["batteryheadtext"]; ok {
		batteryheadtext = safk.(string)
	}
	if safk, ok = user_config["phone"].(map[string]any)["batteryheadtextpower"]; ok {
		batteryheadtextpower = safk.(string)
	}

	if safk, ok = user_config["vrchatoscport"]; ok {
		config.VRChatOSCPort = int(safk.(float64))
	}

	// 启动键盘监听
	go keyboard.KeyboardInput()
	if safk, ok = user_config["music"].(map[string]any)["status"]; ok && safk.(bool) {
		safk, ok = user_config["music"].(map[string]any)["showartist"]
		music_showartist = ok && safk.(bool)
		go music.Init()
	}
	if safk, ok = user_config["iot"].(map[string]any)["status"]; ok && safk.(bool) {
		if safk, ok = user_config["iot"].(map[string]any)["port"]; ok {
			config.IotPort = int(safk.(float64))
		}
		go iot.Init()
	}

	update_messages()
}

func update_messages() {
	for {
		// 检查是否在60秒内有按键 且 当前处于VRChat
		if time.Now().Unix()-keyboard.KeyboardInputSecond < typingtimeout && screen.GetCurrentAppName() == "VRChat" {
			time.Sleep(1 * time.Second)
			continue
		}
		if iot.OffComputerTimeout {
			client := osc.NewClient("localhost", 9000)
			msg := osc.NewMessage("/chatbox/input")
			msg.Append(" - WAITING FOR SHUTDOWN - \n" + fmt.Sprintf("%d", 120-(time.Now().Unix()-iot.OffComputerTimeouts)) + "Seconds")
			msg.Append(true)
			msg.Append(false)
			client.Send(msg)
			switch value := (iot.OffComputerTimeouts + 120 - time.Now().Unix()); {
			case value >= 60:
				time.Sleep(15 * time.Second)
			default:
				time.Sleep(3 * time.Second)

			}
			continue
		}

		if autooffcomputer && time.Now().Unix()-config.KeyboardInputTime >= autooffcomputertimeout {
			iot.OffComputer("长时间无操作超时自动关机")
			continue
		}

		// 长时间无操作 -> 显示AFK
		if afk && time.Now().Unix()-config.KeyboardInputTime >= afk_timeout {
			client := osc.NewClient("localhost", config.VRChatOSCPort)
			message := " - AFK - \n" + secondsToHHMMSS(time.Now().Unix()-config.KeyboardInputTime)
			if config.KeyboardInputTime == 0 {
				message = " - OSC Start - "
			}
			msg := osc.NewMessage("/chatbox/input")
			msg.Append(message)
			msg.Append(true)
			msg.Append(false)
			client.Send(msg)

			if config.KeyboardInputTime == 0 {
				config.KeyboardInputTime = time.Now().Unix()
				time.Sleep(5 * time.Second)
			} else {
				time.Sleep(time.Duration(afk_clock) * time.Second)
			}
			continue
		}

		// -------- 构建当前状态消息（仅桌面应用）--------
		message := ""
		var i = 0 // 记录是否已有内容，用于添加换行

		if phone {
			if usenetworkdevice {
				adb.GetData_SelfUse()
			} else {
				adb.GetData()
			}

			if config.Phone.Screen {
				if config.Phone.AppName != "" {
					message += phone_applicationheadtext + config.Phone.AppName
					i++
				}
			}
		}

		last_desktop_app = screen.GetCurrentAppName()
		config.TopApplicationTitle = last_desktop_app
		if last_desktop_app != "" && last_desktop_app != "VRChat" {
			// 先检查是否需要隐藏该应用
			if !isHidden(last_desktop_app) {
				// 获取替换后的显示文本（若没有匹配的规则，则默认显示完整标题）
				displayText := applyTitleRules(last_desktop_app)
				if displayText == "" {
					// 没有匹配任何 title 规则，使用原始格式
					displayText = applicationheadtext + last_desktop_app
				} else {
					// 规则已给出完整显示内容，若需要可添加前缀
					// 这里假设规则 to 返回的就是最终显示内容（支持 [💻] 等前缀）
				}
				if i > 0 {
					message += "\n"
				}
				message += displayText
				i++
			}
		}

		if phone {
			if config.Phone.Battery <= 20 && config.Phone.Battery > 0 {
				if i > 0 {
					message += "\n"
				}
				if config.Phone.Power {
					message += batteryheadtextpower + strconv.Itoa(config.Phone.Battery) + "%"
				} else {
					message += batteryheadtext + strconv.Itoa(config.Phone.Battery) + "%"
				}
				i++
			}
		}

		if config.MusicPlaying && config.MusicTitle != "" {
			if i > 0 {
				message += "\n"
			}
			message += music_headtext + config.MusicTitle
			if music_showartist && config.MusicArtist != "" {
				message += music_delimiter + config.MusicArtist
			}
			message += "\n"
		}

		// 发送消息
		if message != "" {
			if config.ContinueNextMessage {
				config.ContinueNextMessage = false
				continue
			}
			// 隐私替换
			message = applyHideText(message)

			client := osc.NewClient("localhost", config.VRChatOSCPort)
			msg := osc.NewMessage("/chatbox/input")
			msg.Append(message)
			msg.Append(true)
			msg.Append(false)
			client.Send(msg)
		} else {
			log.Println(last_desktop_app)
		}

		time.Sleep(time.Duration(config.WaitTime) * time.Second)
	}
}

// isHidden 判断窗口标题是否应被隐藏
func isHidden(title string) bool {
	for _, rule := range hideRules {
		re, err := regexp.Compile(rule.From)
		if err != nil {
			continue
		}
		matches := re.FindStringSubmatch(title)
		if matches == nil {
			continue
		}
		if checkRuleConditions(rule.Rule, matches) {
			return true
		}
	}
	return false
}

func applyHideText(title string) string {
	for _, rule := range hidetextRules {
		re, err := regexp.Compile(rule.From)
		if err != nil {
			continue
		}
		matches := re.FindStringSubmatch(title)
		if matches == nil {
			continue
		}
		// 使用 to 字段进行捕获组替换
		title = re.ReplaceAllString(title, rule.To)
	}
	return title
}

// applyTitleRules 应用替换规则，返回替换后的显示文本
// 如果没有匹配规则，返回空字符串（表示使用默认显示）
func applyTitleRules(title string) string {
	for _, rule := range titleRules {
		re, err := regexp.Compile(rule.From)
		if err != nil {
			continue
		}
		matches := re.FindStringSubmatch(title)
		if matches == nil {
			continue
		}
		if !checkRuleConditions(rule.Rule, matches) {
			continue
		}
		// 使用 to 字段进行捕获组替换
		return replaceCaptures(rule.To, matches)
	}
	return ""
}

// checkRuleConditions 检查规则的条件是否满足
// rule: 键为 $1, $2... 的捕获组，值为期望的字符串
func checkRuleConditions(conditions map[string]string, matches []string) bool {
	if len(conditions) == 0 {
		return true // 没有条件，视为满足
	}
	for key, expected := range conditions {
		// 从 $1 提取索引
		idxStr := strings.TrimPrefix(key, "$")
		idx, err := strconv.Atoi(idxStr)
		if err != nil || idx < 1 || idx >= len(matches) {
			return false
		}
		if matches[idx] != expected {
			return false
		}
	}
	return true
}

// replaceCaptures 将模板字符串中的 $1, $2 等替换为对应的捕获组内容
func replaceCaptures(template string, matches []string) string {
	result := template
	// 从 $1 开始替换到最大捕获组
	for i := 1; i < len(matches); i++ {
		placeholder := fmt.Sprintf("$%d", i)
		result = strings.ReplaceAll(result, placeholder, matches[i])
	}
	return result
}

// loadRules 从文件读取规则，每行一个 JSON 对象
func loadRules(path string) []Rule {
	var rules []Rule
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("Warning: cannot read rules file %s: %v\n", path, err)
		return rules
	}
	if err := json.Unmarshal(data, &rules); err != nil {
		log.Printf("Warning: invalid JSON array in %s: %v\n", path, err)
	}
	return rules
}

func secondsToHHMMSS(seconds int64) string {
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, secs)
}
