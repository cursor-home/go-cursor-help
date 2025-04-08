// 配置包，负责管理Cursor应用程序的配置
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// StorageConfig 表示存储配置的结构体
type StorageConfig struct {
	// Mac机器ID，用于telemetry遥测
	TelemetryMacMachineId string `json:"telemetry.macMachineId"`
	// 机器ID，用于telemetry遥测
	TelemetryMachineId    string `json:"telemetry.machineId"`
	// 设备ID，用于telemetry遥测
	TelemetryDevDeviceId  string `json:"telemetry.devDeviceId"`
	// SQM ID，用于telemetry遥测
	TelemetrySqmId        string `json:"telemetry.sqmId"`
	// 最后修改时间
	LastModified          string `json:"lastModified"`
	// 配置版本
	Version               string `json:"version"`
}

// Manager 处理配置操作的管理器
type Manager struct {
	// 配置文件路径
	configPath string
	// 互斥锁，保证并发安全
	mu         sync.RWMutex
}

// NewManager 创建一个新的配置管理器
func NewManager(username string) (*Manager, error) {
	// 获取配置文件路径
	configPath, err := getConfigPath(username)
	if err != nil {
		return nil, fmt.Errorf("failed to get config path: %w", err)
	}
	return &Manager{configPath: configPath}, nil
}

// ReadConfig 读取现有配置
func (m *Manager) ReadConfig() (*StorageConfig, error) {
	// 获取读锁
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 读取配置文件
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		// 如果文件不存在，返回nil
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// 解析JSON到配置结构体
	var config StorageConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// SaveConfig 保存配置
func (m *Manager) SaveConfig(config *StorageConfig, readOnly bool) error {
	// 获取写锁
	m.mu.Lock()
	defer m.mu.Unlock()

	// 确保父目录存在
	if err := os.MkdirAll(filepath.Dir(m.configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// 准备更新后的配置
	updatedConfig := m.prepareUpdatedConfig(config)

	// 写入配置
	if err := m.writeConfigFile(updatedConfig, readOnly); err != nil {
		return err
	}

	return nil
}

// prepareUpdatedConfig 合并现有配置与更新
func (m *Manager) prepareUpdatedConfig(config *StorageConfig) map[string]interface{} {
	// 读取现有配置
	originalFile := make(map[string]interface{})
	if data, err := os.ReadFile(m.configPath); err == nil {
		json.Unmarshal(data, &originalFile)
	}

	// 更新字段
	originalFile["telemetry.sqmId"] = config.TelemetrySqmId
	originalFile["telemetry.macMachineId"] = config.TelemetryMacMachineId
	originalFile["telemetry.machineId"] = config.TelemetryMachineId
	originalFile["telemetry.devDeviceId"] = config.TelemetryDevDeviceId
	originalFile["lastModified"] = time.Now().UTC().Format(time.RFC3339)
	// originalFile["version"] = "1.0.1"

	return originalFile
}

// writeConfigFile 处理配置文件的原子写入
func (m *Manager) writeConfigFile(config map[string]interface{}, readOnly bool) error {
	// 带缩进格式化JSON
	content, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// 写入临时文件
	tmpPath := m.configPath + ".tmp"
	if err := os.WriteFile(tmpPath, content, 0666); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	// 设置最终权限
	fileMode := os.FileMode(0666)
	if readOnly {
		fileMode = 0444
	}

	// 设置文件权限
	if err := os.Chmod(tmpPath, fileMode); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to set temporary file permissions: %w", err)
	}

	// 原子重命名
	if err := os.Rename(tmpPath, m.configPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename file: %w", err)
	}

	// 同步目录
	if dir, err := os.Open(filepath.Dir(m.configPath)); err == nil {
		defer dir.Close()
		dir.Sync()
	}

	return nil
}

// getConfigPath 返回配置文件的路径
func getConfigPath(username string) (string, error) {
	var configDir string
	// 根据操作系统确定配置目录路径
	switch runtime.GOOS {
	case "windows":
		configDir = filepath.Join(os.Getenv("APPDATA"), "Cursor", "User", "globalStorage")
	case "darwin":
		configDir = filepath.Join("/Users", username, "Library", "Application Support", "Cursor", "User", "globalStorage")
	case "linux":
		configDir = filepath.Join("/home", username, ".config", "Cursor", "User", "globalStorage")
	default:
		return "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
	return filepath.Join(configDir, "storage.json"), nil
}
