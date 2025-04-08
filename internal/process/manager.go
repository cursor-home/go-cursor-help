// 进程包，负责处理与操作系统进程相关的操作
package process

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Config 保存进程管理器的配置
type Config struct {
	// 终止进程的最大尝试次数
	MaxAttempts     int           
	// 重试之间的延迟时间
	RetryDelay      time.Duration 
	// 要查找的进程名称模式
	ProcessPatterns []string      
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		MaxAttempts: 3,
		RetryDelay:  2 * time.Second,
		ProcessPatterns: []string{
			"Cursor.exe", // Windows可执行文件
			"Cursor ",    // Linux/macOS可执行文件，带空格
			"cursor ",    // Linux/macOS可执行文件，小写带空格
			"cursor",     // Linux/macOS可执行文件，小写
			"Cursor",     // Linux/macOS可执行文件
			"*cursor*",   // 任何包含cursor的进程
			"*Cursor*",   // 任何包含Cursor的进程
		},
	}
}

// Manager 处理进程相关操作的管理器
type Manager struct {
	// 配置信息
	config *Config
	// 日志记录器
	log    *logrus.Logger
}

// NewManager 创建一个新的进程管理器，可选配置和日志记录器
func NewManager(config *Config, log *logrus.Logger) *Manager {
	if config == nil {
		config = DefaultConfig()
	}
	if log == nil {
		log = logrus.New()
	}
	return &Manager{
		config: config,
		log:    log,
	}
}

// IsCursorRunning 检查是否有Cursor进程当前正在运行
func (m *Manager) IsCursorRunning() bool {
	processes, err := m.getCursorProcesses()
	if err != nil {
		m.log.Warn("Failed to get Cursor processes:", err)
		return false
	}
	return len(processes) > 0
}

// KillCursorProcesses 尝试终止所有运行中的Cursor进程
func (m *Manager) KillCursorProcesses() error {
	for attempt := 1; attempt <= m.config.MaxAttempts; attempt++ {
		processes, err := m.getCursorProcesses()
		if err != nil {
			return fmt.Errorf("failed to get processes: %w", err)
		}

		if len(processes) == 0 {
			return nil
		}

		// 在Windows上先尝试优雅关闭
		if runtime.GOOS == "windows" {
			for _, pid := range processes {
				exec.Command("taskkill", "/PID", pid).Run()
				time.Sleep(500 * time.Millisecond)
			}
		}

		// 强制终止剩余进程
		remainingProcesses, _ := m.getCursorProcesses()
		for _, pid := range remainingProcesses {
			m.killProcess(pid)
		}

		time.Sleep(m.config.RetryDelay)

		if processes, _ := m.getCursorProcesses(); len(processes) == 0 {
			return nil
		}
	}

	return nil
}

// getCursorProcesses 返回运行中的Cursor进程的PID列表
func (m *Manager) getCursorProcesses() ([]string, error) {
	cmd := m.getProcessListCommand()
	if cmd == nil {
		return nil, fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute command: %w", err)
	}

	return m.parseProcessList(string(output)), nil
}

// getProcessListCommand 根据操作系统返回适当的列出进程的命令
func (m *Manager) getProcessListCommand() *exec.Cmd {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("tasklist", "/FO", "CSV", "/NH")
	case "darwin":
		return exec.Command("ps", "-ax")
	case "linux":
		return exec.Command("ps", "-A")
	default:
		return nil
	}
}

// parseProcessList 从进程列表输出中提取Cursor进程的PID
func (m *Manager) parseProcessList(output string) []string {
	var processes []string
	for _, line := range strings.Split(output, "\n") {
		lowerLine := strings.ToLower(line)

		// 忽略自身进程
		if m.isOwnProcess(lowerLine) {
			continue
		}

		if pid := m.findCursorProcess(line, lowerLine); pid != "" {
			processes = append(processes, pid)
		}
	}
	return processes
}

// isOwnProcess 检查进程是否属于本应用程序
func (m *Manager) isOwnProcess(line string) bool {
	return strings.Contains(line, "cursor-id-modifier") ||
		strings.Contains(line, "cursor-helper")
}

// findCursorProcess 检查进程行是否匹配Cursor模式并返回其PID
func (m *Manager) findCursorProcess(line, lowerLine string) string {
	for _, pattern := range m.config.ProcessPatterns {
		if m.matchPattern(lowerLine, strings.ToLower(pattern)) {
			return m.extractPID(line)
		}
	}
	return ""
}

// matchPattern 检查一行是否匹配模式，支持通配符
func (m *Manager) matchPattern(line, pattern string) bool {
	switch {
	case strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*"):
		// *text* 模式：包含text
		search := pattern[1 : len(pattern)-1]
		return strings.Contains(line, search)
	case strings.HasPrefix(pattern, "*"):
		// *text 模式：以text结尾
		return strings.HasSuffix(line, pattern[1:])
	case strings.HasSuffix(pattern, "*"):
		// text* 模式：以text开头
		return strings.HasPrefix(line, pattern[:len(pattern)-1])
	default:
		// text 模式：完全匹配
		return line == pattern
	}
}

// extractPID 根据操作系统格式从进程列表行中提取进程ID
func (m *Manager) extractPID(line string) string {
	switch runtime.GOOS {
	case "windows":
		parts := strings.Split(line, ",")
		if len(parts) >= 2 {
			return strings.Trim(parts[1], "\"")
		}
	case "darwin", "linux":
		parts := strings.Fields(line)
		if len(parts) >= 1 {
			return parts[0]
		}
	}
	return ""
}

// killProcess 通过PID强制终止进程
func (m *Manager) killProcess(pid string) error {
	cmd := m.getKillCommand(pid)
	if cmd == nil {
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
	return cmd.Run()
}

// getKillCommand 根据操作系统返回适当的终止进程的命令
func (m *Manager) getKillCommand(pid string) *exec.Cmd {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("taskkill", "/F", "/PID", pid)
	case "darwin", "linux":
		return exec.Command("kill", "-9", pid)
	default:
		return nil
	}
}
