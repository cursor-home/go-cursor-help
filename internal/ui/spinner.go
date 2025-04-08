// UI包
package ui

import (
	"fmt"
	"sync"
	"time"

	"github.com/fatih/color"
)

// SpinnerConfig 定义旋转器的配置
type SpinnerConfig struct {
	// 旋转器动画帧
	Frames []string
	// 帧更新之间的延迟
	Delay  time.Duration
}

// DefaultSpinnerConfig 返回默认旋转器配置
func DefaultSpinnerConfig() *SpinnerConfig {
	return &SpinnerConfig{
		Frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		Delay:  100 * time.Millisecond,
	}
}

// Spinner 表示一个进度旋转器
type Spinner struct {
	// 配置信息
	config  *SpinnerConfig
	// 显示的消息
	message string
	// 当前帧索引
	current int
	// 是否处于活动状态
	active  bool
	// 停止信号通道
	stopCh  chan struct{}
	// 同步互斥锁
	mu      sync.RWMutex
}

// NewSpinner 创建一个具有给定配置的新旋转器
func NewSpinner(config *SpinnerConfig) *Spinner {
	if config == nil {
		config = DefaultSpinnerConfig()
	}
	return &Spinner{
		config: config,
		stopCh: make(chan struct{}),
	}
}

// 状态管理

// SetMessage 设置旋转器消息
func (s *Spinner) SetMessage(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.message = message
}

// IsActive 返回旋转器当前是否处于活动状态
func (s *Spinner) IsActive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.active
}

// 控制方法

// Start 开始旋转器动画
func (s *Spinner) Start() {
	s.mu.Lock()
	if s.active {
		s.mu.Unlock()
		return
	}
	s.active = true
	s.mu.Unlock()

	go s.run()
}

// Stop 停止旋转器动画
func (s *Spinner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.active {
		return
	}

	s.active = false
	close(s.stopCh)
	s.stopCh = make(chan struct{})
	fmt.Print("\r") // 清除旋转器行
}

// 内部方法

// run 运行旋转器动画循环
func (s *Spinner) run() {
	ticker := time.NewTicker(s.config.Delay)
	defer ticker.Stop()

	cyan := color.New(color.FgCyan, color.Bold)
	message := s.message

	// 打印初始状态
	fmt.Printf("\r %s %s", cyan.Sprint(s.config.Frames[0]), message)

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.mu.RLock()
			if !s.active {
				s.mu.RUnlock()
				return
			}
			frame := s.config.Frames[s.current%len(s.config.Frames)]
			s.current++
			s.mu.RUnlock()

			fmt.Printf("\r %s", cyan.Sprint(frame))
			fmt.Printf("\033[%dG%s", 4, message) // 移动光标并打印消息
		}
	}
}
