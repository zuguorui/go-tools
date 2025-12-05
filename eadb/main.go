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
Notice: for commands that may affect multiple devices, you will be prompted to select target device(s).
Notice: <*keyword*> means CONTAINS match surrounded by *. package name means exact match.
command:
    setting
        open system setting app
    launcher
        open launcher
    packages
        list all installed packages
    app-info <*keyword*|package name>
        get app info for a package name (e.g., app-info com.example.app)
    screenshot <file>
        take a screenshot and save to local file
    screenrecord <file> [-duration <seconds>]
      record screen to local file, optional duration in seconds, up to 180 seconds
    uninstall <*keyword*|package name>
        uninstall app(s) matching the keyword in package name
    clear-data <*keyword*|package name>
        clear app data for app(s) matching the keyword in package name
    force-stop <*keyword*|package name>
        force stop app(s) matching the keyword in package name
    start <*keyword*|package name>
        start app matching the keyword in package name
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
	case "start":
		if len(os.Args) < 3 {
			fmt.Println("Please provide keyword to match packages for start.")
			return
		}
		execStartApp(os.Args[2])
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
func selectDevice(single bool) []string {

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

	if single {
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
func selectPackage(device string, keyword *string, single bool) ([]string, error) {
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

	if len(packageList) == 1 {
		return packageList, nil
	}

	var filterdPackageList []string
	if keyword == nil {
		filterdPackageList = packageList
	} else {
		finalKeyword := strings.ToLower(*keyword)
		if strings.HasPrefix(finalKeyword, "*") && strings.HasSuffix(finalKeyword, "*") {
			trimmedKeyword := strings.Trim(finalKeyword, "*")
			for _, pkg := range packageList {
				if strings.Contains(pkg, trimmedKeyword) {
					filterdPackageList = append(filterdPackageList, pkg)
				}
			}
		} else {
			for _, pkg := range packageList {
				if pkg == finalKeyword {
					filterdPackageList = append(filterdPackageList, pkg)
				}
			}
		}
	}

	if len(filterdPackageList) > 1 {
		fmt.Println("Multiple matching packages found:")
		if single {
			for i, pkg := range filterdPackageList {
				fmt.Printf("[%d] %s\n", i+1, pkg)
			}
			fmt.Print("Please enter the index to select package: ")
			var input int
			fmt.Scanln(&input)
			if input <= 0 || input > len(filterdPackageList) {
				fmt.Println("Invalid index, operation cancelled.")
				return nil, fmt.Errorf("invalid index")
			}
			return []string{filterdPackageList[input-1]}, nil
		} else {
			for i, pkg := range filterdPackageList {
				fmt.Printf("[%d] %s\n", i+1, pkg)
			}
			fmt.Print("Please enter the index to select package, or type \"all\" to select all matched packages: ")
			var input string
			fmt.Scanln(&input)
			input = strings.TrimSpace(strings.ToLower(input))
			if input == "all" {
				return filterdPackageList, nil
			} else {
				index, err := strconv.Atoi(input)
				if err != nil || index <= 0 || index > len(filterdPackageList) {
					fmt.Println("Invalid index, operation cancelled.")
					return nil, fmt.Errorf("invalid index")
				}
				return []string{filterdPackageList[index-1]}, nil
			}
		}
	} else {
		return filterdPackageList, nil
	}
}

func execDeleteApp(keyword string) {
	if strings.HasPrefix(keyword, "*") && strings.HasSuffix(keyword, "*") {
		devices := selectDevice(true)
		if len(devices) == 0 {
			return
		}
		device := devices[0]
		packages, err := selectPackage(device, &keyword, false)
		if err != nil {
			fmt.Println("failed to select packages:", err)
			return
		}

		for _, pkg := range packages {
			execStdCommand(device, "uninstall", pkg)
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
	devices := selectDevice(true)
	if len(devices) == 0 {
		return
	}
	device := devices[0]

	packages, err := selectPackage(device, &keyword, true)
	if err != nil || len(packages) == 0 {
		fmt.Println("failed to select packages:", err)
		return
	}

	execStdCommand(device, "shell", "dumpsys", "package", packages[0])
}

func execClearAppData(keyword string) {
	if strings.HasPrefix(keyword, "*") && strings.HasSuffix(keyword, "*") {
		devices := selectDevice(true)
		if len(devices) == 0 {
			return
		}
		device := devices[0]
		packages, err := selectPackage(device, &keyword, false)
		if err != nil || len(packages) == 0 {
			fmt.Println("failed to select packages:", err)
			return
		}

		for _, pkg := range packages {
			execStdCommand(device, "shell", "pm", "clear", pkg)
		}
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
		packages, err := selectPackage(device, &keyword, false)
		if err != nil || len(packages) == 0 {
			fmt.Println("failed to select packages:", err)
			return
		}

		for _, pkg := range packages {
			execStdCommand(device, "shell", "am", "force-stop", pkg)
		}

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

func execStartApp(keyword string) {
	if strings.HasPrefix(keyword, "*") && strings.HasSuffix(keyword, "*") {
		devices := selectDevice(true)
		if len(devices) == 0 {
			return
		}
		device := devices[0]
		packages, err := selectPackage(device, &keyword, true)
		if err != nil || len(packages) == 0 {
			fmt.Println("failed to get packages:", err)
			return
		}

		execStdCommand(device, "shell", "monkey", "-p", packages[0], "-c", "android.intent.category.LAUNCHER", "1")
	} else {
		devices := selectDevice(false)
		if len(devices) == 0 {
			return
		}
		for _, device := range devices {
			execStdCommand(device, "shell", "monkey", "-p", keyword, "-c", "android.intent.category.LAUNCHER", "1")
		}
	}
}
