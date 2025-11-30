package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const helpText = `This is enhanced adb tool.
Usage: eadb <command> [args...]
command:
  setting                   open system setting app
  launcher                  open launcher
  packages                  list all installed packages
  app-info <keyword>        get app info for a package name contains keyword (e.g., app-info com.example.app)
  screenshot <file>         take a screenshot and save to local file
  screenrecord <file> [-duration <seconds>]     record screen to local file, optional duration in seconds, up to 180 seconds
  uninstall <keyword>       uninstall app(s) matching the keyword in package name
  clear-data <keyword>      clear app data for app(s) matching the keyword in package name
  force-stop <keyword>      force stop app(s) matching the keyword in package name
other commands will be transmited to adb as it is.
`

func main() {
	if len(os.Args) < 2 {
		printHelp()
		return
	}

	// 处理自定义命令
	cmd := os.Args[1]

	switch cmd {
	case "setting":
		execOpenSetting()
	case "launcher":
		execOpenLauncher()
	case "packages":
		execListPackages()
	case "app-info":
		if len(os.Args) < 3 {
			fmt.Println("Please provide package name or keyword to match packages for app-info.")
			return
		}
		execGetAppInfo(os.Args[2])
	case "screenshot":
		if len(os.Args) < 3 {
			fmt.Println("Please provide local file path for screenshot.")
			return
		}
		execScreenShot(os.Args[2])
	case "screenrecord":
		if len(os.Args) < 3 {
			fmt.Println("Please provide local file path for screenrecord.")
			return
		}
		var duration *int = nil
		if len(os.Args) >= 5 && os.Args[3] == "-duration" {
			dur, err := strconv.Atoi(os.Args[4])
			if err == nil {
				duration = &dur
			} else {
				fmt.Println("Invalid duration value.")
				return
			}
			if *duration > 180 {
				fmt.Println("Duration exceeds maximum limit of 180 seconds.")
				return
			}
			if *duration <= 0 {
				fmt.Println("Duration must be a positive integer.")
				return
			}
		}
		execScreenRecord(os.Args[2], duration)
	case "uninstall":
		if len(os.Args) < 3 {
			fmt.Println("Please provide keyword to match packages for uninstall.")
			return
		}
		execDeleteApp(os.Args[2])
	case "clear-data":
		if len(os.Args) < 3 {
			fmt.Println("Please provide keyword to match packages for clear-data.")
			return
		}
		execClearAppData(os.Args[2])
	case "force-stop":
		if len(os.Args) < 3 {
			fmt.Println("Please provide keyword to match packages for force-stop.")
			return
		}
		execForceStopApp(os.Args[2])
	default:
		execMultiDeviceStdCommand(os.Args[1:]...)
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
func selectDevice(onlySingle bool) []string {

	devices := getDevices()
	if len(devices) == 0 {
		fmt.Println("No devices connected.")
		return []string{}
	}
	if len(devices) == 1 {
		return devices
	}

	fmt.Println("More than one devices found:")
	for i, d := range devices {
		fmt.Printf("[%d] %s\n", i+1, d)
	}

	if onlySingle {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Input index to select a device: ")
		for {
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)
			idx := -1
			fmt.Sscanf(input, "%d", &idx)
			if idx > 0 && idx <= len(devices) {
				target := devices[idx-1]
				fmt.Printf("Selected device: %s\n", target)
				return []string{target}
			}
			fmt.Println("Invalid index, try again.")
		}
	} else {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Input index to select a device, or type \"all\" to select all devices: ")
		for {
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(strings.ToLower(input))
			if input == "all" {
				return devices
			} else {
				idx := -1
				fmt.Sscanf(input, "%d", &idx)
				if idx > 0 && idx <= len(devices) {
					target := devices[idx-1]
					fmt.Printf("Selected device: %s\n", target)
					return []string{target}
				}
				fmt.Println("Invalid index, try again.")
			}
		}
	}
}

func execStdCommand(device string, args ...string) {
	execCommand(os.Stdout, os.Stdin, os.Stderr, device, args...)
}

func execCommand(outputStream io.Writer, inputStream io.Reader, errorStream io.Writer, device string, args ...string) {
	fullArgs := append([]string{"-s", device}, args...)
	fmt.Println("Executing adb command for device ", device, ": adb", fullArgs)
	adbCmd := exec.Command("adb", fullArgs...)
	adbCmd.Stdout = outputStream
	adbCmd.Stderr = errorStream
	adbCmd.Stdin = inputStream
	err := adbCmd.Run()
	if err != nil {
		fmt.Println("Error:", err)
	}
}

func execMultiDeviceStdCommand(args ...string) {
	devices := selectDevice(false)
	if len(devices) == 0 {
		return
	}
	for _, device := range devices {
		execStdCommand(device, args...)
	}
}

func execRootStdCommand(device string, args ...string) error {
	return execRootCommand(os.Stdout, os.Stderr, device, args...)
}

func execRootCommand(outputStream io.Writer, errorStream io.Writer, device string, args ...string) error {
	// 启动 adb shell
	cmd := exec.Command("adb", "-s", device, "shell")
	cmd.Stdout = outputStream
	cmd.Stderr = errorStream

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	// 进入 su
	stdin.Write([]byte("su\n"))

	// 执行命令
	commandStr := strings.Join(args, " ")
	stdin.Write([]byte(commandStr + "\n"))

	// 退出 su
	stdin.Write([]byte("exit\n"))

	// 退出 adb shell
	stdin.Write([]byte("exit\n"))
	stdin.Close()

	return cmd.Wait()
}

func execOpenSetting() {
	devices := selectDevice(true)
	if len(devices) == 0 {
		return
	}
	for _, device := range devices {
		execStdCommand(device, "shell", "am", "start", "-a", "android.settings.SETTINGS")
	}
}

func execOpenLauncher() {
	devices := selectDevice(true)
	if len(devices) == 0 {
		return
	}
	for _, device := range devices {
		execStdCommand(device, "shell", "am", "start", "-a", "android.intent.action.MAIN", "-c", "android.intent.category.HOME")
	}
}

func execListPackages() {
	devices := selectDevice(false)
	if len(devices) == 0 {
		return
	}
	for _, device := range devices {
		execStdCommand(device, "shell", "pm", "list", "packages")
	}
}

func execScreenShot(localPath string) {
	devices := selectDevice(true)
	if len(devices) == 0 {
		return
	}
	for _, device := range devices {
		func() {
			path := fmt.Sprintf("%s_%s.png", localPath, device)
			file, err := os.Create(path)
			if err != nil {
				fmt.Println("Error creating file:", err)
				return
			}
			defer file.Close()
			execCommand(file, os.Stdin, os.Stderr, device, "exec-out", "screencap", "-p")
		}()
	}
}

func execScreenRecord(localPath string, duration *int) {
	devices := selectDevice(true)
	if len(devices) == 0 {
		return
	}
	for _, device := range devices {
		func() {
			if duration != nil {
				execStdCommand(device, "shell", fmt.Sprintf("screenrecord --time-limit %d /sdcard/temp_screenrecord.mp4", *duration))
			} else {
				execStdCommand(device, "shell", "screenrecord /sdcard/temp_screenrecord.mp4")
			}
			path := fmt.Sprintf("%s_%s.mp4", localPath, device)
			execStdCommand(device, "pull", "/sdcard/temp_screenrecord.mp4", path)
			execStdCommand(device, "shell", "rm", "/sdcard/temp_screenrecord.mp4")
		}()
	}
}

/*
获取包名。若keyword为nil，则获取所有包名。如果keyword以*开始或结束，则以contains匹配。否则精确匹配。
*/
func getPackages(device string, keyword *string) ([]string, error) {
	// 1. 获取包列表
	cmd := exec.Command("adb", "-s", device, "shell", "pm", "list", "packages")
	outBytes, err := cmd.Output()
	if err != nil {
		fmt.Println("list packages failed:", err)
		return nil, err
	}
	// 2. 处理输出
	var packageList []string
	for _, line := range strings.Split(string(outBytes), "\n") {
		if strings.HasPrefix(line, "package:") {
			packageList = append(packageList, strings.TrimSpace(strings.TrimPrefix(line, "package:")))
		}
	}

	if keyword == nil {
		return packageList, nil
	}

	finalKeyword := strings.ToLower(*keyword)

	var result []string
	if strings.HasPrefix(finalKeyword, "*") && strings.HasSuffix(finalKeyword, "*") {
		trimmedKeyword := strings.Trim(finalKeyword, "*")
		for _, pkg := range packageList {
			if strings.Contains(pkg, trimmedKeyword) {
				result = append(result, pkg)
			}
		}
	} else {
		for _, pkg := range packageList {
			if pkg == finalKeyword {
				result = append(result, pkg)
			}
		}
	}
	return result, nil
}

func execDeleteApp(keyword string) {
	if strings.HasPrefix(keyword, "*") && strings.HasSuffix(keyword, "*") {
		devices := selectDevice(true)
		if len(devices) == 0 {
			return
		}
		device := devices[0]
		packages, err := getPackages(device, &keyword)
		if err != nil {
			fmt.Println("failed to get packages:", err)
			return
		}

		// 3. 匹配结果处理
		if len(packages) == 0 {
			fmt.Println("no matching packages found.")
			return
		} else if len(packages) == 1 {
			fmt.Printf("Found one matching package: %s. Confirm uninstall? (y/N): ", packages[0])
			var input string
			fmt.Scanln(&input)
			if strings.ToLower(input) == "y" {
				execStdCommand(device, "uninstall", packages[0])
			} else {
				fmt.Println("uninstall cancelled.")
			}
			return
		} else {
			fmt.Println("Multiple matching packages found:")
			for i, pkg := range packages {
				fmt.Printf("[%d] %s\n", i+1, pkg)
			}
			fmt.Print("Please enter the index to uninstall, or type \"all\" to uninstall all matched package: ")
			var input string
			fmt.Scanln(&input)
			input = strings.TrimSpace(strings.ToLower(input))
			if input == "all" {
				for _, pkg := range packages {
					execStdCommand(device, "uninstall", pkg)
				}
			} else {
				index, err := strconv.Atoi(input)
				if err != nil || index <= 0 || index > len(packages) {
					fmt.Println("Invalid index, uninstall cancelled.")
					return
				}
				execStdCommand(device, "uninstall", packages[index-1])
			}
		}
	} else {
		devices := selectDevice(false)
		if len(devices) == 0 {
			return
		}
		for _, device := range devices {
			execStdCommand(device, "uninstall", keyword)
		}
	}

}

func execGetAppInfo(keyword string) {
	devices := selectDevice(false)
	if len(devices) == 0 {
		return
	}
	device := devices[0]

	packages, err := getPackages(device, &keyword)
	if err != nil {
		fmt.Println("failed to get packages:", err)
		return
	}

	// 3. 匹配结果处理
	if len(packages) == 0 {
		fmt.Println("no matching packages found.")
	} else if len(packages) == 1 {
		execStdCommand(device, "shell", "dumpsys", "package", packages[0])
	} else {
		fmt.Println("Multiple matching packages found:")
		for i, pkg := range packages {
			fmt.Printf("[%d] %s\n", i+1, pkg)
		}
		fmt.Print("Please enter the index to get app info: ")
		var input int
		fmt.Scanln(&input)
		if input <= 0 || input > len(packages) {
			fmt.Println("Invalid index, operation cancelled.")
			return
		}
		execStdCommand(device, "shell", "dumpsys", "package", packages[input-1])
	}
}

func execClearAppData(keyword string) {
	if strings.HasPrefix(keyword, "*") && strings.HasSuffix(keyword, "*") {
		devices := selectDevice(true)
		if len(devices) == 0 {
			return
		}
		device := devices[0]
		packages, err := getPackages(device, &keyword)
		if err != nil {
			fmt.Println("failed to get packages:", err)
			return
		}

		// 3. 匹配结果处理
		if len(packages) == 0 {
			fmt.Println("no matching packages found.")
			return
		} else if len(packages) == 1 {
			fmt.Printf("Found one matching package: %s. Confirm clear app data? (y/N): ", packages[0])
			var input string
			fmt.Scanln(&input)
			if strings.ToLower(input) == "y" {
				execStdCommand(device, "shell", "pm", "clear", packages[0])
			} else {
				fmt.Println("clear app data cancelled.")
			}
			return
		} else {
			fmt.Println("Multiple matching packages found:")
			for i, pkg := range packages {
				fmt.Printf("[%d] %s\n", i+1, pkg)
			}
			fmt.Print("Please enter the index to clear app data, or type \"all\" to clear app data for all matched package: ")
			var input string
			fmt.Scanln(&input)
			input = strings.TrimSpace(strings.ToLower(input))
			if input == "all" {
				for _, pkg := range packages {
					execStdCommand(device, "shell", "pm", "clear", pkg)
				}
			} else {
				index, err := strconv.Atoi(input)
				if err != nil || index <= 0 || index > len(packages) {
					fmt.Println("Invalid index, operation cancelled.")
					return
				}
				execStdCommand(device, "shell", "pm", "clear", packages[index-1])
			}
		}
		return
	} else {
		devices := selectDevice(false)
		if len(devices) == 0 {
			return
		}
		for _, device := range devices {
			execStdCommand(device, "shell", "pm", "clear", keyword)
		}
	}
}

func execForceStopApp(keyword string) {
	if strings.HasPrefix(keyword, "*") && strings.HasSuffix(keyword, "*") {
		devices := selectDevice(true)
		if len(devices) == 0 {
			return
		}
		device := devices[0]
		packages, err := getPackages(device, &keyword)
		if err != nil {
			fmt.Println("failed to get packages:", err)
			return
		}

		// 3. 匹配结果处理
		if len(packages) == 0 {
			fmt.Println("no matching packages found.")
			return
		} else if len(packages) == 1 {
			fmt.Printf("Found one matching package: %s. Confirm force stop app? (y/N): ", packages[0])
			var input string
			fmt.Scanln(&input)
			if strings.ToLower(input) == "y" {
				execStdCommand(device, "shell", "am", "force-stop", packages[0])
			} else {
				fmt.Println("force stop app cancelled.")
			}
			return
		} else {
			fmt.Println("Multiple matching packages found:")
			for i, pkg := range packages {
				fmt.Printf("[%d] %s\n", i+1, pkg)
			}
			fmt.Print("Please enter the index to force stop app, or type \"all\" to force stop all matched package: ")
			var input string
			fmt.Scanln(&input)
			input = strings.TrimSpace(strings.ToLower(input))
			if input == "all" {
				for _, pkg := range packages {
					execStdCommand(device, "shell", "am", "force-stop", pkg)
				}
			} else {
				index, err := strconv.Atoi(input)
				if err != nil || index <= 0 || index > len(packages) {
					fmt.Println("Invalid index, operation cancelled.")
					return
				}
				execStdCommand(device, "shell", "am", "force-stop", packages[index-1])
			}
		}
		return
	} else {
		devices := selectDevice(false)
		if len(devices) == 0 {
			return
		}
		for _, device := range devices {
			execStdCommand(device, "shell", "am", "force-stop", keyword)
		}
	}
}
