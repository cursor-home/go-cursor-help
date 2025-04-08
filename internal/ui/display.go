// UI包，提供用户界面相关功能
package ui

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/fatih/color"
)

// Display 处理终端输出的UI操作
type Display struct {
	// 进度旋转器
	spinner *Spinner
}

// NewDisplay 创建一个新的显示实例，可选提供旋转器
func NewDisplay(spinner *Spinner) *Display {
	if spinner == nil {
		spinner = NewSpinner(nil)
	}
	return &Display{spinner: spinner}
}

// 终端操作

// ClearScreen 根据操作系统清除终端屏幕
func (d *Display) ClearScreen() error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "cls")
	default:
		cmd = exec.Command("clear")
	}
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

// 进度指示器

// ShowProgress 显示带有旋转器的进度消息
func (d *Display) ShowProgress(message string) {
	d.spinner.SetMessage(message)
	d.spinner.Start()
}

// StopProgress 停止进度旋转器
func (d *Display) StopProgress() {
	d.spinner.Stop()
}

// 消息显示

// ShowSuccess 以绿色显示成功消息
func (d *Display) ShowSuccess(messages ...string) {
	green := color.New(color.FgGreen)
	for _, msg := range messages {
		green.Println(msg)
	}
}

// ShowInfo 以青色显示信息消息
func (d *Display) ShowInfo(message string) {
	cyan := color.New(color.FgCyan)
	cyan.Println(message)
}

// ShowError 以红色显示错误消息
func (d *Display) ShowError(message string) {
	red := color.New(color.FgRed)
	red.Println(message)
}

// ShowPrivilegeError 显示权限错误消息及操作指导
func (d *Display) ShowPrivilegeError(messages ...string) {
	red := color.New(color.FgRed, color.Bold)
	yellow := color.New(color.FgYellow)

	// 主要错误消息
	red.Println(messages[0])
	fmt.Println()

	// 附加指导说明
	for _, msg := range messages[1:] {
		if strings.Contains(msg, "%s") {
			exe, _ := os.Executable()
			yellow.Printf(msg+"\n", exe)
		} else {
			yellow.Println(msg)
		}
	}
}

// ShowLogo 显示应用程序的Logo
func (d *Display) ShowLogo() {
	fmt.Println(GetLogo())
}
