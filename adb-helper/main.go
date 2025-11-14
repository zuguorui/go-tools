package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const helpText = `Usage: adb-helper <command> [args...]
command:
  go-setting 				open setting app
  go-launcher  				open launcher
  packages     				list all installed packages
  app-info <package>     	get app info for a package (e.g., app-info com.example.app)
  screenshot <file>   		take a screenshot and save to local file
  screen-record <file>   	record screen and save to /sdcard/<file> on device
other commands will be transmited to adb as it is.
`

func main() {
	if len(os.Args) < 2 {
		printHelp()
		return
	}

	// 获取设备列表
	devices := getDevices()
	if len(devices) == 0 {
		fmt.Println("No devices connected.")
		return
	}

	// 选择设备
	device := selectDevice(devices)

	var cmdArgs []string
	// 处理自定义命令
	if len(os.Args) == 2 && os.Args[1] == "go-setting" {
		cmdArgs = []string{"shell", "am", "start", "-a", "android.settings.SETTINGS"}
	} else if len(os.Args) == 2 && os.Args[1] == "go-launcher" {
		cmdArgs = []string{"shell", "am", "start", "-a", "android.intent.action.MAIN", "-c", "android.intent.category.HOME"}
	} else if len(os.Args) == 2 && os.Args[1] == "packages" {
		cmdArgs = []string{"shell", "pm", "list", "packages"}
	} else if len(os.Args) == 3 && os.Args[1] == "app-info" {
		cmdArgs = []string{"shell", "dumpsys", "package", os.Args[2]}
	} else if len(os.Args) == 3 && os.Args[1] == "screenshot" {
		cmdArgs = []string{"exec-out", "screencap", "-p >", os.Args[2]}
	} else {
		cmdArgs = os.Args[1:]
	}

	// 构建 adb 命令
	fullArgs := append([]string{"-s", device}, cmdArgs...)
	fmt.Println("fullArgs: ", fullArgs)
	adbCmd := exec.Command("adb", fullArgs...)
	adbCmd.Stdout = os.Stdout
	adbCmd.Stderr = os.Stderr
	adbCmd.Stdin = os.Stdin

	if err := adbCmd.Run(); err != nil {
		fmt.Println("Error:", err)
	}
}

func printHelp() {
	fmt.Println(helpText)
}

// 获取 adb devices
func getDevices() []string {
	out, err := exec.Command("adb", "devices").Output()
	if err != nil {
		fmt.Println("Error running adb:", err)
		return nil
	}

	lines := strings.Split(string(out), "\n")
	devices := []string{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "List of devices") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 && parts[1] == "device" {
			devices = append(devices, parts[0])
		}
	}
	return devices
}

// 让用户选择设备
func selectDevice(devices []string) string {
	if len(devices) == 1 {
		return devices[0]
	}

	fmt.Println("More than one devices found:")
	for i, d := range devices {
		fmt.Printf("[%d] %s\n", i+1, d)
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Input index to select a device: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		idx := -1
		fmt.Sscanf(input, "%d", &idx)
		if idx > 0 && idx <= len(devices) {
			target := devices[idx-1]
			fmt.Printf("Selected device: %s\n", target)
			return target
		}
		fmt.Println("Invalid index, try again.")
	}
}
