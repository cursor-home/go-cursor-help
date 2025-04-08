// idgen包，提供安全的ID生成功能
package idgen

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
)

// Generator 处理机器和设备的安全ID生成
type Generator struct {
	// 字节缓冲池，用于减少内存分配
	bufferPool sync.Pool
}

// NewGenerator 创建一个新的ID生成器
func NewGenerator() *Generator {
	return &Generator{
		bufferPool: sync.Pool{
			New: func() interface{} {
				return make([]byte, 64)
			},
		},
	}
}

// ID生成的常量
const (
	// 机器ID前缀
	machineIDPrefix = "auth0|user_"
	// UUID格式模板
	uuidFormat      = "%s-%s-%s-%s-%s"
)

// generateRandomHex 生成指定长度的随机十六进制字符串
func (g *Generator) generateRandomHex(length int) (string, error) {
	// 从池中获取缓冲区
	buffer := g.bufferPool.Get().([]byte)
	defer g.bufferPool.Put(buffer)

	// 生成随机字节
	if _, err := rand.Read(buffer[:length]); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return hex.EncodeToString(buffer[:length]), nil
}

// GenerateMachineID 生成带有auth0|user_前缀的新机器ID
func (g *Generator) GenerateMachineID() (string, error) {
	// 生成64字符的十六进制随机部分
	randomPart, err := g.generateRandomHex(32) 
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x%s", []byte(machineIDPrefix), randomPart), nil
}

// GenerateMacMachineID 生成新的64字节MAC机器ID
func (g *Generator) GenerateMacMachineID() (string, error) {
	// 生成64字符的十六进制
	return g.generateRandomHex(32) 
}

// GenerateDeviceID 以UUID格式生成新的设备ID
func (g *Generator) GenerateDeviceID() (string, error) {
	id, err := g.generateRandomHex(16)
	if err != nil {
		return "", err
	}
	// 按照UUID格式拼接字符串
	return fmt.Sprintf(uuidFormat,
		id[0:8], id[8:12], id[12:16], id[16:20], id[20:32]), nil
}

// GenerateSQMID 以带花括号的UUID格式生成新的SQM ID
func (g *Generator) GenerateSQMID() (string, error) {
	id, err := g.GenerateDeviceID()
	if err != nil {
		return "", err
	}
	// 在UUID两侧添加花括号
	return fmt.Sprintf("{%s}", id), nil
}

// ValidateID 验证各种ID类型的格式
func (g *Generator) ValidateID(id string, idType string) bool {
	switch idType {
	case "machineID", "macMachineID":
		// 机器ID应该是64个十六进制字符
		return len(id) == 64 && isHexString(id)
	case "deviceID":
		// 设备ID应该是有效的UUID格式
		return isValidUUID(id)
	case "sqmID":
		// SQM ID应该是带花括号的UUID
		if len(id) < 2 || id[0] != '{' || id[len(id)-1] != '}' {
			return false
		}
		return isValidUUID(id[1 : len(id)-1])
	default:
		return false
	}
}

// 辅助函数

// isHexString 检查字符串是否为有效的十六进制字符串
func isHexString(s string) bool {
	_, err := hex.DecodeString(s)
	return err == nil
}

// isValidUUID 检查字符串是否为有效的UUID格式
func isValidUUID(uuid string) bool {
	// UUID应该有36个字符
	if len(uuid) != 36 {
		return false
	}
	// 检查每个字符是否符合UUID格式规范
	for i, r := range uuid {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if r != '-' {
				return false
			}
			continue
		}
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}
