package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// 调用参数
var (
	showHelp            *bool
	modeLite            *bool
	modeGeneral         *bool
	codesign            *bool
	delUselessFiles     *bool
	skipDeleteipa       *bool
	skipDeleteAllTarget *bool
)

func init() {
	showHelp = flag.Bool("h", false, "Show help information")
	modeLite = flag.Bool("mode-lite", false, "Required selection mode")
	modeGeneral = flag.Bool("mode-general", false, "Required selection mode")
	codesign = flag.Bool("codesign", false, "")
	delUselessFiles = flag.Bool("del-useless-files", false, "Delete TargetApp and dSYM that are repeatedly packaged into the product")
	skipDeleteipa = flag.Bool("skip-delete-ipa", false, "Skip deleting the ipa in the TargetApp directory")
	skipDeleteAllTarget = flag.Bool("skip-delete-all-target", false, "Skip deleting all temporarily generated files in the TargetApp directory")
}

// 扫描targetapp下是否有符合条件的app或ipa，如果是ipa则让后续函数去解压，然后把符合条件的xxx.app这个名字返回
func getAppFromTargetApp(targetAppPath string) string {
	// Count apps and ipas
	appCount := 0
	ipaCount := 0
	selectedApp := ""

	// 检查TargetApp 这个目录是否存在
	if _, err := os.Stat(targetAppPath); os.IsNotExist(err) {
		fmt.Println("Error: Unable to find target directory. Do not delete the folder generated by the template.")
		os.Exit(0)
	}
	items, err := os.ReadDir(targetAppPath)
	if err != nil {
		fmt.Printf("Error: The TargetApp directory cannot be found. Make sure TargetApp exists in the Xcode target name directory.")
		os.Exit(1)
	}
	for _, item := range items {
		itemPath := item.Name()
		if item.IsDir() {
			if strings.HasSuffix(item.Name(), ".app") {
				appCount++
				selectedApp = itemPath
			}
		} else {
			if strings.HasSuffix(item.Name(), ".ipa") {
				ipaCount++
				if appCount == 0 {
					selectedApp = itemPath
				}
			}
		}
	}
	if appCount > 1 || ipaCount > 1 {
		fmt.Println("Error: Multiple executables found, you should only keep one")
		os.Exit(1)
	} else if appCount == 1 {

	} else if ipaCount == 1 {
		selectedApp = unzipiPA(filepath.Join(targetAppPath, selectedApp))
	} else {
		fmt.Println("Error: You should drag the application you want to execute into the TargetApp directory")
		os.Exit(1)
	}
	return selectedApp
}

// 如果有sdym符号文件就一并拷贝走
func getSDYMFile(sdymPath string) string {
	if _, err := os.Stat(sdymPath); os.IsNotExist(err) {
		fmt.Println("Warn: Unable to find dSYM directory. Do not delete the folder generated by the template. Ignore processing of dSYM files")
		return ""
	}
	items, err := os.ReadDir(sdymPath)
	if err != nil {
		fmt.Printf("Warn: The dSYM directory cannot be found. Make sure dSYM exists in the Xcode target name directory.")
	}
	count := 0
	sdymName := ""
	for _, item := range items {
		if strings.HasSuffix(item.Name(), ".dSYM") {
			count++
			sdymName = item.Name()
		}
	}
	if count > 1 {
		fmt.Printf("Warn: Multiple dSYM files were found. Please delete useless files. This compilation will ignore the dSYM step.")
		return ""
	}
	return sdymName
}

// 解压ipa 并且把ipa里面的app转移到targetapp目录下
func unzipiPA(ipaPath string) string {
	dir := filepath.Dir(ipaPath)
	cmd := exec.Command("/usr/bin/unzip", "-o", ipaPath, "-d", dir)
	_, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Error: Failed to decompress ipa. Manually unzip the ipa, drag and drop the AppName.app into the TargetApp directory and try again")
		os.Exit(1)
	}
	// 判断payload是否合法
	payloadPath := filepath.Join(dir, "Payload")
	payloadPathInfo, err := os.Stat(payloadPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("Error: Payload Directory does not exist. ipa format is illegal")
			os.Exit(1)
		} else {
			fmt.Println("Error: ", err)
			os.Exit(1)
		}
	}
	if payloadPathInfo.IsDir() {
		// 获取文件夹下的条目
		entries, err := os.ReadDir(payloadPath)
		if err != nil {
			fmt.Println("Error: reading directory - ", err)
			os.Exit(1)
		}
		for _, entry := range entries {
			if entry.IsDir() && strings.HasSuffix(entry.Name(), ".app") {
				cmd := exec.Command("mv", filepath.Join(payloadPath, entry.Name()), dir)
				err := cmd.Run()
				if err != nil {
					fmt.Println("Error: moving Payload directory - ", err)
					os.Exit(0)
				} else {
					if !*skipDeleteAllTarget {
						if !*skipDeleteipa {
							err := os.Remove(ipaPath)
							if err != nil {
								fmt.Println("Warn: ", "Deletion of ipa under TargetApp failed")
							}
						}
						err = os.Remove(payloadPath)
						if err != nil {
							fmt.Println("Warn: ", "Failed to delete the ipa decompression package under TargetApp")
						}
					}
					return entry.Name()
				}
			}
		}
		fmt.Println("Error: ", "The structure after decompressing the ipa is illegal")
	} else {
		fmt.Println("Error: ", "The structure after decompressing the ipa is illegal. Do not place useless files in the TargetApp directory.")
	}
	os.Exit(1)
	return ""
}

func codesignApp(identity string, appPath string) {
	//TODO 删除PlugIns
	exec.Command("rm", "-rf", filepath.Join(appPath, "PlugIns")).Run()
	// 签名本体
	cmd := exec.Command("/usr/bin/codesign", "-f", "-s", identity, appPath)
	_, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Error: signing program - ", err.Error())
		os.Exit(1)
	}
	frameworkPath := filepath.Join(appPath, "Frameworks")
	// 获取文件的相关信息
	info, err := os.Stat(frameworkPath)
	if err == nil {
		if info.IsDir() {
			entries, err := os.ReadDir(frameworkPath)
			if err != nil {
				fmt.Println("Error: Failed to open Frameworks directory - ", err)
				os.Exit(1)
			}
			for _, entry := range entries {
				if entry.IsDir() {
					exec.Command("/usr/bin/codesign", "-f", "-s", identity, "--deep", filepath.Join(frameworkPath, entry.Name())).Run()
				}
			}
		}
	}
}

func delelteUselessFiles(appPath string) {
	items, err := os.ReadDir(appPath)
	if err != nil {
		fmt.Printf("Warn: The generated app directory cannot be read, and the deletion step is skipped.")
	} else {
		for _, item := range items {
			if item.IsDir() {
				if strings.HasSuffix(item.Name(), ".app") || strings.HasSuffix(item.Name(), ".app.dSYM") {
					exec.Command("rm", "-r", filepath.Join(appPath, item.Name())).Run()
				}
			}
		}
	}
}

func main() {
	flag.Parse()
	if len(os.Args) == 1 || *showHelp {
		// flag.Usage()
		fmt.Println("Usage:")
		flag.VisitAll(func(f *flag.Flag) {
			fmt.Printf("  -%-22s %s\n", f.Name, f.Usage)
		})
		os.Exit(0)
	}

	// 取出来需要的一些环境变量
	builtProductsDir := os.Getenv("BUILT_PRODUCTS_DIR")
	targetName := os.Getenv("TARGET_NAME")
	srcRoot := os.Getenv("SRCROOT")
	productBundleIdentifier := os.Getenv("PRODUCT_BUNDLE_IDENTIFIER")
	expandedCodeSignIdentity := os.Getenv("EXPANDED_CODE_SIGN_IDENTITY")
	TargetAppPath := filepath.Join(srcRoot, targetName, "TargetApp")
	buildAppPath := filepath.Join(builtProductsDir, targetName+".app")

	if *delUselessFiles {
		delelteUselessFiles(buildAppPath)
		os.Exit(0)
	}

	if *codesign {
		codesignApp(expandedCodeSignIdentity, buildAppPath)
		os.Exit(0)
	}

	appFolder := getAppFromTargetApp(TargetAppPath)
	targetInfoPlist := filepath.Join(srcRoot, targetName, "Info.plist")

	cmd := exec.Command("/usr/libexec/PlistBuddy", "-c", "Print CFBundleExecutable", targetInfoPlist)
	currentExecutableBytes, _ := cmd.CombinedOutput()
	currentExecutable := strings.TrimSpace(string(currentExecutableBytes))

	copyAppPath := filepath.Join(TargetAppPath, appFolder)

	exec.Command("rm", "-r", buildAppPath).Run()
	exec.Command("cp", "-r", copyAppPath, buildAppPath).Run()

	cmd = exec.Command("/usr/libexec/PlistBuddy", "-c", "Print CFBundleExecutable", filepath.Join(copyAppPath, "Info.plist"))
	targetExecutableBytes, _ := cmd.CombinedOutput()
	targetExecutable := strings.TrimSpace(string(targetExecutableBytes))

	if currentExecutable != targetExecutable {
		exec.Command("cp", filepath.Join(copyAppPath, "Info.plist"), targetInfoPlist).Run()
	}

	exec.Command("/usr/libexec/PlistBuddy", "-c", "Set :CFBundleIdentifier "+productBundleIdentifier, targetInfoPlist).Run()
	exec.Command("/usr/libexec/PlistBuddy", "-c", "Delete UISupportedDevices", targetInfoPlist).Run()
	exec.Command("cp", targetInfoPlist, filepath.Join(buildAppPath, "Info.plist")).Run()

	// lite 模式不生成dylib，不需要签名dylib
	if !*modeLite {
		appBinary := targetExecutable
		// Install using optool
		exec.Command("/opt/AppleDebugger/bin/optool", "install", "-p", "@rpath/lib"+targetName+"Dylib.dylib", "-t", filepath.Join(buildAppPath, appBinary)).Run()
		// exec.Command("/opt/AppleDebugger/bin/optool", "install", "-p", "@executable_path/lib"+targetName+"Dylib.dylib", "-t", filepath.Join(buildAppPath, appBinary)).Run()
	}

	dSYMPath := filepath.Join(srcRoot, targetName, "dSYM")
	sdymFile := getSDYMFile(dSYMPath)
	if sdymFile != "" {
		newDSYMPath := filepath.Join(builtProductsDir, targetName+".app.dSYM")
		exec.Command("rm", "-r", newDSYMPath).Run()
		exec.Command("cp", "-R", filepath.Join(dSYMPath, sdymFile), newDSYMPath).Run()
	}
}
