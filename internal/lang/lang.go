// 语言包，提供多语言支持功能
package lang

import (
	"os"
	"os/exec"
	"strings"
	"sync"
)

// Language 表示支持的语言代码类型
type Language string

const (
	// CN 表示中文语言
	CN Language = "cn"
	// EN 表示英文语言
	EN Language = "en"
)

// TextResource 包含所有可翻译的文本资源
type TextResource struct {
	// 成功消息
	SuccessMessage string
	RestartMessage string

	// 进度消息
	ReadingConfig     string
	GeneratingIds     string
	CheckingProcesses string
	ClosingProcesses  string
	ProcessesClosed   string
	PleaseWait        string

	// 错误消息
	ErrorPrefix    string
	PrivilegeError string

	// 指令提示
	RunAsAdmin         string
	RunWithSudo        string
	SudoExample        string
	PressEnterToExit   string
	SetReadOnlyMessage string

	// 信息消息
	ConfigLocation string
}

var (
	// 当前语言设置
	currentLanguage     Language
	// 确保语言检测只执行一次的同步机制
	currentLanguageOnce sync.Once
	// 保护语言变量的互斥锁
	languageMutex       sync.RWMutex
)

// GetCurrentLanguage 返回当前语言，如果尚未设置则自动检测
func GetCurrentLanguage() Language {
	currentLanguageOnce.Do(func() {
		currentLanguage = detectLanguage()
	})

	languageMutex.RLock()
	defer languageMutex.RUnlock()
	return currentLanguage
}

// SetLanguage 设置当前语言
func SetLanguage(lang Language) {
	languageMutex.Lock()
	defer languageMutex.Unlock()
	currentLanguage = lang
}

// GetText 返回当前语言的文本资源
func GetText() TextResource {
	return texts[GetCurrentLanguage()]
}

// detectLanguage 检测系统语言
func detectLanguage() Language {
	// 首先检查环境变量
	if isChineseEnvVar() {
		return CN
	}

	// 然后检查特定操作系统的区域设置
	if isWindows() {
		if isWindowsChineseLocale() {
			return CN
		}
	} else if isUnixChineseLocale() {
		return CN
	}

	return EN
}

// isChineseEnvVar 检查环境变量是否表明系统使用中文
func isChineseEnvVar() bool {
	for _, envVar := range []string{"LANG", "LANGUAGE", "LC_ALL"} {
		if lang := os.Getenv(envVar); lang != "" && strings.Contains(strings.ToLower(lang), "zh") {
			return true
		}
	}
	return false
}

// isWindows 判断当前操作系统是否为Windows
func isWindows() bool {
	return os.Getenv("OS") == "Windows_NT"
}

// isWindowsChineseLocale 检查Windows系统是否使用中文区域设置
func isWindowsChineseLocale() bool {
	// 检查Windows UI文化设置
	cmd := exec.Command("powershell", "-Command",
		"[System.Globalization.CultureInfo]::CurrentUICulture.Name")
	output, err := cmd.Output()
	if err == nil && strings.HasPrefix(strings.ToLower(strings.TrimSpace(string(output))), "zh") {
		return true
	}

	// 检查Windows区域设置
	cmd = exec.Command("wmic", "os", "get", "locale")
	output, err = cmd.Output()
	return err == nil && strings.Contains(string(output), "2052")
}

// isUnixChineseLocale 检查Unix系统是否使用中文区域设置
func isUnixChineseLocale() bool {
	cmd := exec.Command("locale")
	output, err := cmd.Output()
	return err == nil && strings.Contains(strings.ToLower(string(output)), "zh_cn")
}

// texts 包含所有翻译文本
var texts = map[Language]TextResource{
	CN: {
		// 成功消息
		SuccessMessage: "[√] 配置文件已成功更新！",
		RestartMessage: "[!] 请手动重启 Cursor 以使更新生效",

		// 进度消息
		ReadingConfig:     "正在读取配置文件...",
		GeneratingIds:     "正在生成新的标识符...",
		CheckingProcesses: "正在检查运行中的 Cursor 实例...",
		ClosingProcesses:  "正在关闭 Cursor 实例...",
		ProcessesClosed:   "所有 Cursor 实例已关闭",
		PleaseWait:        "请稍候...",

		// 错误消息
		ErrorPrefix:    "程序发生严重错误: %v",
		PrivilegeError: "\n[!] 错误：需要管理员权限",

		// 指令提示
		RunAsAdmin:         "请右键点击程序，选择「以管理员身份运行」",
		RunWithSudo:        "请使用 sudo 命令运行此程序",
		SudoExample:        "示例: sudo %s",
		PressEnterToExit:   "\n按回车键退出程序...",
		SetReadOnlyMessage: "设置 storage.json 为只读模式, 这将导致 workspace 记录信息丢失等问题",

		// 信息消息
		ConfigLocation: "配置文件位置:",
	},
	EN: {
		// 成功消息
		SuccessMessage: "[√] Configuration file updated successfully!",
		RestartMessage: "[!] Please restart Cursor manually for changes to take effect",

		// 进度消息
		ReadingConfig:     "Reading configuration file...",
		GeneratingIds:     "Generating new identifiers...",
		CheckingProcesses: "Checking for running Cursor instances...",
		ClosingProcesses:  "Closing Cursor instances...",
		ProcessesClosed:   "All Cursor instances have been closed",
		PleaseWait:        "Please wait...",

		// 错误消息
		ErrorPrefix:    "Program encountered a serious error: %v",
		PrivilegeError: "\n[!] Error: Administrator privileges required",

		// 指令提示
		RunAsAdmin:         "Please right-click and select 'Run as Administrator'",
		RunWithSudo:        "Please run this program with sudo",
		SudoExample:        "Example: sudo %s",
		PressEnterToExit:   "\nPress Enter to exit...",
		SetReadOnlyMessage: "Set storage.json to read-only mode, which will cause issues such as lost workspace records",

		// 信息消息
		ConfigLocation: "Config file location:",
	},
}
