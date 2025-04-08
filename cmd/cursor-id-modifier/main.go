package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/yuaotian/go-cursor-help/internal/config"
	"github.com/yuaotian/go-cursor-help/internal/lang"
	"github.com/yuaotian/go-cursor-help/internal/process"
	"github.com/yuaotian/go-cursor-help/internal/ui"
	"github.com/yuaotian/go-cursor-help/pkg/idgen"
)

// 全局变量定义
var (
	// version: 程序版本号，默认为"dev"，在构建时可能会被替换为实际版本号
	version = "dev"
	// setReadOnly: 命令行标志，用于设置storage.json文件为只读模式
	// 当设置为true时，保存配置后会将文件权限设置为只读
	setReadOnly = flag.Bool("r", false, "set storage.json to read-only mode")
	// showVersion: 命令行标志，用于显示程序版本信息
	// 当设置为true时，程序会显示版本号并退出
	showVersion = flag.Bool("v", false, "show version information")
	// log: 全局日志记录器实例，使用logrus库提供高级日志功能
	// 用于记录程序运行过程中的各种信息、警告和错误
	log = logrus.New()
)

// main: 程序入口函数
// 负责协调整个程序的执行流程，包括初始化、权限检查、配置处理等
func main() {
	// 2025-04-08 11:35:26 by cc 捕获了个寂寞？？？
	// 设置错误恢复机制，防止程序因panic而崩溃
	setupErrorRecovery()
	// 解析并处理命令行参数
	handleFlags()
	// 配置日志记录器的格式和级别
	setupLogger()

	// 获取当前用户名，用于定位配置文件
	username := getCurrentUser()
	log.Debug("Running as user:", username)

	// 初始化各个组件
	// display: 用户界面显示组件，负责输出信息到控制台
	display := ui.NewDisplay(nil)
	// configManager: 配置管理器，负责读取和保存配置文件
	configManager := initConfigManager(username)
	// generator: ID生成器，用于生成各种唯一标识符
	generator := idgen.NewGenerator()
	// processManager: 进程管理器，用于管理Cursor进程
	processManager := process.NewManager(nil, log)

	// 检查并处理程序运行权限，确保有足够权限修改配置文件
	if err := handlePrivileges(display); err != nil {
		return
	}

	// 设置显示界面，清屏并显示程序logo
	setupDisplay(display)

	// 获取当前语言的文本资源，用于多语言支持
	text := lang.GetText()

	// 处理Cursor进程，确保在修改配置前关闭所有Cursor实例
	if err := handleCursorProcesses(display, processManager); err != nil {
		return
	}

	// 读取现有配置，获取当前的Cursor配置信息
	oldConfig := readExistingConfig(display, configManager, text)
	// 生成新的配置，包括新的机器ID、设备ID等
	newConfig := generateNewConfig(display, generator, oldConfig, text)

	// 保存新配置到storage.json文件
	if err := saveConfiguration(display, configManager, newConfig); err != nil {
		return
	}

	// 显示操作完成的消息，提示用户重启Cursor
	showCompletionMessages(display)

	// 如果不是自动化模式（通常是权限提升后的进程），则等待用户按Enter键退出
	// 这样用户可以看到程序的输出结果
	if os.Getenv("AUTOMATED_MODE") != "1" {
		waitExit()
	}
}

// setupErrorRecovery: 设置错误恢复机制
// 使用defer和recover捕获可能发生的panic，防止程序崩溃
// 当panic发生时，会记录错误信息并打印堆栈跟踪，然后等待用户按键退出
func setupErrorRecovery() {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("Panic recovered: %v\n", r)
			debug.PrintStack()
			waitExit()
		}
	}()
}

// handleFlags: 处理命令行参数
// 解析命令行标志，并根据标志执行相应操作
// 如果设置了showVersion标志，则显示版本信息并退出程序
func handleFlags() {
	flag.Parse()
	if *showVersion {
		fmt.Printf("Cursor ID Modifier v%s\n", version)
		os.Exit(0)
	}
}

// setupLogger: 设置日志记录器的格式和级别
// 配置logrus日志记录器，设置输出格式为包含完整时间戳的文本格式
// 设置日志级别为Info，记录信息、警告和错误消息
func setupLogger() {
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:          true, // 显示完整时间戳
		DisableLevelTruncation: true, // 不截断日志级别文本
		PadLevelText:           true, // 对齐日志级别文本
	})
	log.SetLevel(logrus.InfoLevel) // 设置日志级别为Info
}

// getCurrentUser: 获取当前用户名
// 首先尝试获取SUDO_USER环境变量（在使用sudo运行时表示实际用户）
// 如果SUDO_USER不存在，则获取当前系统用户
// 返回值:
//   - string: 当前用户名
func getCurrentUser() string {
	// 优先获取SUDO_USER环境变量，这在使用sudo运行程序时很重要
	// 因为我们需要知道实际的用户，而不是root用户
	if username := os.Getenv("SUDO_USER"); username != "" {
		return username
	}

	// 如果SUDO_USER不存在，则获取当前系统用户
	user, err := user.Current()
	if err != nil {
		log.Fatal(err) // 如果获取用户失败，记录错误并终止程序
	}
	return user.Username // 返回用户名
}

// initConfigManager: 初始化配置管理器
// 创建一个新的配置管理器实例，用于读取和保存Cursor的配置文件
// 参数:
//   - username: 用户名，用于定位配置文件路径
//
// 返回值:
//   - *config.Manager: 配置管理器实例
func initConfigManager(username string) *config.Manager {
	configManager, err := config.NewManager(username)
	if err != nil {
		log.Fatal(err) // 如果创建配置管理器失败，记录错误并终止程序
	}
	return configManager // 返回配置管理器实例
}

// handlePrivileges: 处理权限检查
// 检查程序是否具有足够的权限（管理员/root权限）来修改配置文件
// 如果没有足够权限，会尝试提升权限或显示错误消息
// 参数:
//   - display: 用户界面显示组件，用于显示错误消息
//
// 返回值:
//   - error: 如果权限检查失败或权限不足且无法提升，则返回错误
func handlePrivileges(display *ui.Display) error {
	// 检查是否具有管理员/root权限
	isAdmin, err := checkAdminPrivileges()
	if err != nil {
		log.Error(err) // 记录错误
		waitExit()     // 等待用户按键退出
		return err
	}

	// 如果没有管理员/root权限
	if !isAdmin {
		// Windows系统特殊处理，尝试自动提升权限
		if runtime.GOOS == "windows" {
			return handleWindowsPrivileges(display)
		}
		// 非Windows系统显示权限错误消息，提示用户使用sudo运行
		display.ShowPrivilegeError(
			lang.GetText().PrivilegeError,
			lang.GetText().RunWithSudo,
			lang.GetText().SudoExample,
		)
		waitExit()                                   // 等待用户按键退出
		return fmt.Errorf("insufficient privileges") // 返回权限不足错误
	}
	return nil // 权限检查通过，返回nil
}

// handleWindowsPrivileges: 处理Windows系统的权限提升
// 在Windows系统上尝试自动提升程序权限到管理员级别
// 参数:
//   - display: 用户界面显示组件，用于显示错误消息
//
// 返回值:
//   - error: 如果权限提升失败，则返回错误
func handleWindowsPrivileges(display *ui.Display) error {
	// 显示请求管理员权限的消息，根据当前语言选择不同文本
	message := "\nRequesting administrator privileges..."
	if lang.GetCurrentLanguage() == lang.CN {
		message = "\n请求管理员权限..."
	}
	fmt.Println(message)

	// 尝试自我提升权限，启动一个新的具有管理员权限的进程
	if err := selfElevate(); err != nil {
		log.Error(err) // 记录错误
		// 显示权限错误消息，提示用户手动以管理员身份运行
		display.ShowPrivilegeError(
			lang.GetText().PrivilegeError,
			lang.GetText().RunAsAdmin,
			lang.GetText().RunWithSudo,
			lang.GetText().SudoExample,
		)
		waitExit() // 等待用户按键退出
		return err // 返回错误
	}
	return nil // 权限提升成功或已启动新进程，返回nil
}

// setupDisplay: 设置显示界面
// 清屏并显示程序logo，为用户提供清晰的界面
// 参数:
//   - display: 用户界面显示组件
func setupDisplay(display *ui.Display) {
	// 尝试清屏，如果失败则记录警告但继续执行
	if err := display.ClearScreen(); err != nil {
		log.Warn("Failed to clear screen:", err)
	}
	// 显示程序logo
	display.ShowLogo()
	// 打印空行，增加界面可读性
	fmt.Println()
}

// handleCursorProcesses: 处理Cursor进程
// 尝试关闭所有运行中的Cursor进程，确保在修改配置前没有Cursor实例在运行
// 参数:
//   - display: 用户界面显示组件，用于显示进度和错误消息
//   - processManager: 进程管理器，用于管理Cursor进程
//
// 返回值:
//   - error: 如果无法关闭Cursor进程，则返回错误
func handleCursorProcesses(display *ui.Display, processManager *process.Manager) error {
	// 自动化模式下跳过关闭Cursor进程
	// 这通常是在权限提升后的新进程中，避免重复操作
	if os.Getenv("AUTOMATED_MODE") == "1" {
		log.Debug("Running in automated mode, skipping Cursor process closing")
		return nil
	}

	// 显示正在关闭Cursor的进度信息
	display.ShowProgress("Closing Cursor...")
	log.Debug("Attempting to close Cursor processes")

	// 尝试终止所有Cursor进程
	if err := processManager.KillCursorProcesses(); err != nil {
		log.Error("Failed to close Cursor:", err) // 记录错误
		display.StopProgress()                    // 停止进度显示
		// 显示错误消息，提示用户手动关闭Cursor
		display.ShowError("Failed to close Cursor. Please close it manually and try again.")
		waitExit() // 等待用户按键退出
		return err // 返回错误
	}

	// 再次检查是否仍有Cursor进程在运行
	// 这是一个额外的安全检查，确保所有进程都已关闭
	if processManager.IsCursorRunning() {
		log.Error("Cursor processes still detected after closing")
		display.StopProgress() // 停止进度显示
		// 显示错误消息，提示用户手动关闭Cursor
		display.ShowError("Failed to close Cursor completely. Please close it manually and try again.")
		waitExit()                                // 等待用户按键退出
		return fmt.Errorf("cursor still running") // 返回错误
	}

	// 成功关闭所有Cursor进程
	log.Debug("Successfully closed all Cursor processes")
	display.StopProgress() // 停止进度显示
	fmt.Println()          // 打印空行，增加界面可读性
	return nil             // 返回nil表示成功
}

// readExistingConfig: 读取现有配置
// 尝试读取Cursor的现有配置文件，获取当前的配置信息
// 参数:
//   - display: 用户界面显示组件，用于显示进度
//   - configManager: 配置管理器，用于读取配置文件
//   - text: 语言文本资源，用于多语言支持
//
// 返回值:
//   - *config.StorageConfig: 读取到的配置，如果读取失败则返回nil
func readExistingConfig(display *ui.Display, configManager *config.Manager, text lang.TextResource) *config.StorageConfig {
	fmt.Println()                            // 打印空行，增加界面可读性
	display.ShowProgress(text.ReadingConfig) // 显示正在读取配置的进度信息

	// 尝试读取现有配置
	oldConfig, err := configManager.ReadConfig()
	if err != nil {
		log.Warn("Failed to read existing config:", err) // 记录警告
		oldConfig = nil                                  // 如果读取失败，设置为nil
	}

	display.StopProgress() // 停止进度显示
	fmt.Println()          // 打印空行，增加界面可读性
	return oldConfig       // 返回读取到的配置或nil
}

// generateNewConfig: 生成新的配置
// 生成新的Cursor配置，包括新的机器ID、设备ID等
// 参数:
//   - display: 用户界面显示组件，用于显示进度
//   - generator: ID生成器，用于生成各种唯一标识符
//   - oldConfig: 现有配置，用于保留某些值（如SQM ID）
//   - text: 语言文本资源，用于多语言支持
//
// 返回值:
//   - *config.StorageConfig: 生成的新配置
func generateNewConfig(display *ui.Display, generator *idgen.Generator, oldConfig *config.StorageConfig, text lang.TextResource) *config.StorageConfig {
	display.ShowProgress(text.GeneratingIds) // 显示正在生成ID的进度信息
	newConfig := &config.StorageConfig{}     // 创建新的配置对象

	// 生成机器ID
	if machineID, err := generator.GenerateMachineID(); err != nil {
		log.Fatal("Failed to generate machine ID:", err) // 如果生成失败，记录错误并终止程序
	} else {
		newConfig.TelemetryMachineId = machineID // 设置新的机器ID
	}

	// 生成Mac机器ID（特定于macOS的标识符）
	if macMachineID, err := generator.GenerateMacMachineID(); err != nil {
		log.Fatal("Failed to generate MAC machine ID:", err) // 如果生成失败，记录错误并终止程序
	} else {
		newConfig.TelemetryMacMachineId = macMachineID // 设置新的Mac机器ID
	}

	// 生成设备ID
	if deviceID, err := generator.GenerateDeviceID(); err != nil {
		log.Fatal("Failed to generate device ID:", err) // 如果生成失败，记录错误并终止程序
	} else {
		newConfig.TelemetryDevDeviceId = deviceID // 设置新的设备ID
	}

	// 生成或保留SQM ID
	// 如果存在旧配置且SQM ID不为空，则保留原有的SQM ID
	// 否则生成新的SQM ID
	if oldConfig != nil && oldConfig.TelemetrySqmId != "" {
		newConfig.TelemetrySqmId = oldConfig.TelemetrySqmId // 保留原有的SQM ID
	} else if sqmID, err := generator.GenerateSQMID(); err != nil {
		log.Fatal("Failed to generate SQM ID:", err) // 如果生成失败，记录错误并终止程序
	} else {
		newConfig.TelemetrySqmId = sqmID // 设置新的SQM ID
	}

	display.StopProgress() // 停止进度显示
	fmt.Println()          // 打印空行，增加界面可读性
	return newConfig       // 返回生成的新配置
}

// saveConfiguration: 保存配置
// 将新生成的配置保存到Cursor的配置文件中
// 参数:
//   - display: 用户界面显示组件，用于显示进度
//   - configManager: 配置管理器，用于保存配置文件
//   - newConfig: 要保存的新配置
//
// 返回值:
//   - error: 如果保存失败，则返回错误
func saveConfiguration(display *ui.Display, configManager *config.Manager, newConfig *config.StorageConfig) error {
	display.ShowProgress("Saving configuration...") // 显示正在保存配置的进度信息

	// 保存新配置到文件，并根据setReadOnly标志决定是否设置为只读
	if err := configManager.SaveConfig(newConfig, *setReadOnly); err != nil {
		log.Error(err) // 记录错误
		waitExit()     // 等待用户按键退出
		return err     // 返回错误
	}

	display.StopProgress() // 停止进度显示
	fmt.Println()          // 打印空行，增加界面可读性
	return nil             // 返回nil表示成功
}

// showCompletionMessages: 显示完成消息
// 显示操作成功完成的消息，提示用户重启Cursor
// 参数:
//   - display: 用户界面显示组件，用于显示成功和信息消息
func showCompletionMessages(display *ui.Display) {
	// 显示成功消息，包括操作成功和需要重启Cursor的提示
	display.ShowSuccess(lang.GetText().SuccessMessage, lang.GetText().RestartMessage)
	fmt.Println() // 打印空行，增加界面可读性

	// 根据当前语言显示操作完成消息
	message := "Operation completed!"
	if lang.GetCurrentLanguage() == lang.CN {
		message = "操作完成！"
	}
	display.ShowInfo(message) // 显示信息消息
}

// waitExit: 等待用户按下Enter键退出
// 显示提示消息并等待用户按下Enter键，然后程序退出
// 这使用户有时间阅读程序输出的信息
func waitExit() {
	fmt.Print(lang.GetText().PressEnterToExit) // 显示按Enter退出的提示
	os.Stdout.Sync()                           // 刷新标准输出缓冲区，确保消息立即显示
	bufio.NewReader(os.Stdin).ReadString('\n') // 读取用户输入，直到按下Enter键
}

// checkAdminPrivileges: 检查是否具有管理员权限
// 此函数根据不同操作系统检查当前进程是否具有管理员/root权限
// 返回值：
//   - bool: 如果具有管理员权限则为true，否则为false
//   - error: 如果检查过程中发生错误则返回相应错误，否则为nil
func checkAdminPrivileges() (bool, error) {
	switch runtime.GOOS {
	case "windows":
		// Windows系统下的权限检查方法
		// 通过执行"net session"命令来检查权限，此命令只有管理员才能成功执行
		// 如果命令执行成功(返回nil错误)，则表示具有管理员权限
		cmd := exec.Command("net", "session")
		return cmd.Run() == nil, nil

	case "darwin", "linux":
		// macOS和Linux系统下的权限检查方法
		// 通过检查当前用户的UID是否为0(root用户的UID)来确定
		// 获取当前用户信息
		currentUser, err := user.Current()
		if err != nil {
			// 如果获取用户信息失败，返回错误
			return false, fmt.Errorf("failed to get current user: %w", err)
		}
		// 检查UID是否为"0"(root用户)
		return currentUser.Uid == "0", nil

	default:
		// 对于不支持的操作系统，返回错误
		return false, fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// selfElevate: 自我权限提升函数
// 用于将程序提升到管理员/root权限运行
// 此函数根据不同操作系统执行相应的权限提升操作
// 同时设置环境变量以防止提权后的进程再次等待用户输入
// 返回值：
//   - error: 如果权限提升过程中发生错误则返回相应错误，否则为nil
func selfElevate() error {
	// 设置自动化模式环境变量，防止提权后的进程再次等待用户输入
	// 这样可以避免在提权后的进程中再次显示等待用户按Enter退出的提示
	os.Setenv("AUTOMATED_MODE", "1")

	switch runtime.GOOS {
	case "windows":
		// Windows系统下使用"runas"提升权限
		verb := "runas"                        // "runas"是Windows中用于以管理员身份运行程序的命令
		exe, _ := os.Executable()              // 获取当前可执行文件的路径
		cwd, _ := os.Getwd()                   // 获取当前工作目录
		args := strings.Join(os.Args[1:], " ") // 将命令行参数合并为一个字符串，不包括程序名称

		// 创建一个新的命令，通过cmd.exe启动当前程序并提升权限
		// "/C"表示执行完命令后关闭cmd窗口
		// "start"用于启动新进程
		// verb参数指定以管理员身份运行
		cmd := exec.Command("cmd", "/C", "start", verb, exe, args)
		cmd.Dir = cwd    // 设置命令的工作目录，确保在相同的目录下执行
		return cmd.Run() // 执行命令并返回可能的错误

	case "darwin", "linux":
		// macOS和Linux系统下使用sudo提升权限
		exe, err := os.Executable() // 获取当前可执行文件的路径
		if err != nil {
			return err // 如果获取失败，返回错误
		}

		// 创建一个使用sudo的命令，将当前程序及其参数作为sudo的参数
		// append([]string{exe}, os.Args[1:]...) 将可执行文件路径和原始参数组合成新的参数列表
		cmd := exec.Command("sudo", append([]string{exe}, os.Args[1:]...)...)
		// 将标准输入、输出和错误流连接到当前进程的对应流
		cmd.Stdin = os.Stdin   // 允许用户输入sudo密码
		cmd.Stdout = os.Stdout // 显示命令输出，保持用户可以看到程序的输出信息
		cmd.Stderr = os.Stderr // 显示错误信息，确保错误信息能够正确显示给用户
		return cmd.Run()       // 执行命令并返回可能的错误

	default:
		// 对于不支持的操作系统，返回错误
		// 明确指出当前操作系统不受支持
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}
