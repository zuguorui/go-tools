package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const CONFIG_FILE_NAME = "scrcpy_config.txt"
const SCRCPY_DIR = "SCRCPY_DIR"

// 全局 scrcpy 路径
var scrcpyPath string = ""

func main() {
	// 加载配置
	err := loadConfig()
	if err != nil {
		fmt.Println("❌ 加载配置失败: ", err)
		return
	}

	// 获取设备
	devices, err := getADBDevices()
	if err != nil {
		fmt.Println("❌ 获取设备列表失败:", err)
		return
	}

	if len(devices) == 0 {
		fmt.Println("❌ 未检测到任何已连接的 ADB 设备。")
		return
	}

	if len(devices) == 1 {
		fmt.Printf("✅ 检测到单一设备：%s\n", devices[0])
		runScrcpy(devices[0])
		return
	}

	fmt.Println("📱 检测到多个设备：")
	for i, dev := range devices {
		fmt.Printf("[%d] %s\n", i+1, dev)
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("请输入要连接的设备序号（1-%d）: ", len(devices))
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	index, err := strconv.Atoi(input)
	if err != nil || index < 1 || index > len(devices) {
		fmt.Println("❌ 无效输入。")
		return
	}

	device := devices[index-1]
	fmt.Println("🔗 正在连接设备：", device)
	runScrcpy(device)
}

func getConfigPath() string {
	exePath, err := os.Executable()
	if err != nil {
		panic(err)
	}

	exeDir := filepath.Dir(exePath)
	return filepath.Join(exeDir, "scrcpy_config.txt")
}

// 读取配置文件
func loadConfig() error {
	configPath := getConfigPath()
	file, err := os.OpenFile(configPath, os.O_RDWR|os.O_CREATE, 0777)
	if err != nil {
		return fmt.Errorf("无法打开配置文件 %s", configPath)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		strArray := strings.Split(line, "=")
		if len(strArray) != 2 {
			continue
		}
		if strings.Trim(strArray[0], " ") != SCRCPY_DIR {
			continue
		}
		scrcpyPath = strings.Trim(strArray[1], " ")
		fmt.Println("scrcpy路径: ", scrcpyPath)
		break
	}

	if scrcpyPath == "" {
		return fmt.Errorf("未检测到有效的scrcpy路径，请在%s中添加行: %s=your scrcpy dir", configPath, SCRCPY_DIR)
	}

	return nil
}

// 读取 adb devices
func getADBDevices() ([]string, error) {
	cmd := exec.Command("adb", "devices")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	devices := []string{}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasSuffix(line, "\tdevice") {
			parts := strings.Fields(line)
			if len(parts) > 0 {
				devices = append(devices, parts[0])
			}
		}
	}

	return devices, nil
}

// 运行 scrcpy
func runScrcpy(device string) {
	cmd := exec.Command(scrcpyPath+"/scrcpy", "-s", device)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}
