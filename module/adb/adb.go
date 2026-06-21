package adb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
	"vrchat_osc/module/config"
)

const ADB_PATH = "adb"
const JSON_PATH = "./data/androidappkeys.json"
const BAN_PATH = "./data/androidappban.json"

// AppNameItem 随机标题配置项
type AppNameItem struct {
	Content string  `json:"content"`
	P       float64 `json:"p"`
}

// loadAppNameMap 加载包名->应用名的映射表（支持字符串或随机列表）
func loadAppNameMap() (map[string]any, error) {
	data, err := os.ReadFile(JSON_PATH)
	if err != nil {
		return nil, fmt.Errorf("读取应用名映射文件失败: %w", err)
	}

	var mapping map[string]any
	if err := json.Unmarshal(data, &mapping); err != nil {
		return nil, fmt.Errorf("解析JSON失败: %w", err)
	}

	return mapping, nil
}

// loadBanList 加载黑名单包名列表
func loadBanList() (map[string]bool, error) {
	data, err := os.ReadFile(BAN_PATH)
	if err != nil {
		return nil, fmt.Errorf("读取黑名单文件失败: %w", err)
	}

	var packages []string
	if err := json.Unmarshal(data, &packages); err != nil {
		return nil, fmt.Errorf("解析黑名单JSON失败: %w", err)
	}

	banMap := make(map[string]bool)
	for _, pkg := range packages {
		banMap[pkg] = true
	}

	return banMap, nil
}

// isBanned 检查包名是否在黑名单中
func isBanned(packageName string) bool {
	banMap, err := loadBanList()
	if err != nil {
		// 黑名单加载失败，放行所有应用
		return false
	}
	return banMap[packageName]
}

// resolveAppName 根据映射值解析出最终应用名
// value 可能是 string（固定名称）或 []any（随机列表）
func resolveAppName(value any) string {
	switch v := value.(type) {
	case string:
		return v

	case []any:
		var items []AppNameItem
		for _, item := range v {
			itemMap, ok := item.(map[string]any)
			if !ok {
				continue
			}
			content, _ := itemMap["content"].(string)
			p, _ := itemMap["p"].(float64)
			if content != "" && p > 0 {
				items = append(items, AppNameItem{Content: content, P: p})
			}
		}

		if len(items) == 0 {
			return ""
		}

		return randomSelect(items)

	default:
		return ""
	}
}

// randomSelect 按权重随机选择
func randomSelect(items []AppNameItem) string {
	totalWeight := 0.0
	for _, item := range items {
		totalWeight += item.P
	}

	r := rand.Float64() * totalWeight

	cumulative := 0.0
	for _, item := range items {
		cumulative += item.P
		if r < cumulative {
			return item.Content
		}
	}

	return items[0].Content
}

// GetForegroundApp 获取当前前台应用的中文名（从JSON映射表获取）
func GetForegroundApp() (string, error) {
	// 1. 获取当前前台包名
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, ADB_PATH, "shell", "dumpsys", "activity", "recents")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("执行adb命令失败: %w", err)
	}

	re := regexp.MustCompile(`(?s)\*\s+Recent #0:.*?A=\d+:([^}\s]+)`)
	match := re.FindSubmatch(output)
	if match == nil {
		return "", fmt.Errorf("未找到最近的Activity记录")
	}
	packageName := strings.TrimSpace(string(match[1]))

	// 2. 检查黑名单
	if isBanned(packageName) {
		return "", nil
	}
	// 5. 映射表中不存在或解析失败，降级返回包名
	return packageName, nil
}

func GetAppName(packageName string) (string, error) {
	// 3. 加载包名->应用名映射表
	appMap, err := loadAppNameMap()
	if err != nil {
		println("安卓应用包名表出现异常")
		return packageName, nil
	}

	// 4. 查找映射表
	if value, ok := appMap[packageName]; ok {
		resolved := resolveAppName(value)
		if resolved != "" {
			return resolved, nil
		}
	}
	return packageName, nil
}

// IsScreenOn 检测屏幕是否亮起
func IsScreenOn() (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, ADB_PATH, "shell", "dumpsys", "power")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("执行adb命令失败: %w", err)
	}

	output := string(out)
	if strings.Contains(output, "mWakefulness=Awake") {
		return true, nil
	}
	if strings.Contains(output, "mWakefulness=Dreaming") {
		return true, nil
	}
	if strings.Contains(output, "mWakefulness=Asleep") {
		return false, nil
	}
	if strings.Contains(output, "Display Power: state=ON") {
		return true, nil
	}
	if strings.Contains(output, "Display Power: state=OFF") {
		return false, nil
	}

	return false, fmt.Errorf("无法解析屏幕状态")
}

// BatteryInfo 电池信息
type BatteryInfo struct {
	Level      int    // 当前电量（百分比）
	Scale      int    // 最大电量（通常 100）
	Status     int    // 充电状态：1=未知 2=充电中 3=放电中 4=未充电 5=充满
	StatusText string // 状态中文描述
	Plugged    string // 充电类型：AC/USB/Wireless/None
	Temp       int    // 温度（单位：0.1°C）
	Voltage    int    // 电压（mV）
}

// GetBatteryInfo 获取电池完整信息
func GetBatteryInfo() (*BatteryInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, ADB_PATH, "shell", "dumpsys", "battery")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("执行adb命令失败: %w", err)
	}

	output := string(out)
	info := &BatteryInfo{}

	// 提取 level
	if re := regexp.MustCompile(`level:\s*(\d+)`); re.MatchString(output) {
		match := re.FindStringSubmatch(output)
		info.Level, _ = strconv.Atoi(match[1])
	}

	// 提取 scale
	if re := regexp.MustCompile(`scale:\s*(\d+)`); re.MatchString(output) {
		match := re.FindStringSubmatch(output)
		info.Scale, _ = strconv.Atoi(match[1])
	}

	// 提取 status
	if re := regexp.MustCompile(`status:\s*(\d+)`); re.MatchString(output) {
		match := re.FindStringSubmatch(output)
		info.Status, _ = strconv.Atoi(match[1])
	}

	// 提取充电类型
	switch {
	case strings.Contains(output, "AC powered: true"):
		info.Plugged = "AC"
	case strings.Contains(output, "USB powered: true"):
		info.Plugged = "USB"
	case strings.Contains(output, "Wireless powered: true"):
		info.Plugged = "Wireless"
	default:
		info.Plugged = "None"
	}

	// 提取温度（单位 0.1°C）
	if re := regexp.MustCompile(`temperature:\s*(\d+)`); re.MatchString(output) {
		match := re.FindStringSubmatch(output)
		info.Temp, _ = strconv.Atoi(match[1])
	}

	// 提取电压（mV）
	if re := regexp.MustCompile(`voltage:\s*(\d+)`); re.MatchString(output) {
		match := re.FindStringSubmatch(output)
		info.Voltage, _ = strconv.Atoi(match[1])
	}

	// 状态映射
	switch info.Status {
	case 1:
		info.StatusText = "未知"
	case 2:
		info.StatusText = "充电中"
	case 3:
		info.StatusText = "放电中"
	case 4:
		info.StatusText = "未充电"
	case 5:
		info.StatusText = "已充满"
	}

	return info, nil
}

func GetData() {
	v, err := IsScreenOn()
	if err != nil {
		v = false
	}
	config.Phone.Screen = v
	v2, err := GetForegroundApp()
	if err != nil {
		v2 = ""
	}
	config.Phone.AppPackage = v2
	v2, err = GetAppName(v2)
	if err != nil {
		v2 = ""
	}
	config.Phone.AppName = v2
	v3, err := GetBatteryInfo()
	if err != nil {
		v3 = &BatteryInfo{}
	}
	config.Phone.Battery = v3.Level
	config.Phone.Power = v3.Plugged != "None"
}

func GetData_SelfUse() {
	resp, err := http.Get("https://cengtuyin.24h.fyi/api/devices_status")
	if err == nil {
		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err == nil {
			var devices_status = make(map[string]any)
			if json.Unmarshal(data, &devices_status) == nil {
				deviceData, ok := devices_status["RedmiK60Ultra"].(map[string]any)
				screenOn, ok := deviceData["SCREEN"].(bool)
				config.Phone.Screen = ok && screenOn
				appname, _ := deviceData["APP_NAME"].(string)
				config.Phone.AppName = appname
				appname, _ = deviceData["APP"].(string)
				config.Phone.AppPackage = appname
				if isBanned(appname) {
					config.Phone.AppName = ""
				}
				batteryStr, ok := deviceData["BATTERY"].(string)
				sv, _ := strconv.Atoi(batteryStr)
				config.Phone.Battery = sv
				config.Phone.Power = deviceData["POWER"].(bool)
				if deviceData["MUSIC"].(map[string]any)["title"].(string) != "无媒体" {
					config.MusicPlaying = true
					if deviceData["MUSIC"].(map[string]any)["package"].(string) == "tv.danmaku.bili" {
						config.MusicTitle = "VIDEO: " + deviceData["MUSIC"].(map[string]any)["title"].(string)
						config.MusicArtist = deviceData["MUSIC"].(map[string]any)["content"].(string)
					} else {
						config.MusicTitle = "MUSIC: " + deviceData["MUSIC"].(map[string]any)["title"].(string)
						config.MusicArtist = deviceData["MUSIC"].(map[string]any)["content"].(string)
					}
				} else {
					config.MusicPlaying = false
				}
			} else {
				println("无法解析数据")
			}
		} else {
			println("无法读取数据")
		}
	}
}
