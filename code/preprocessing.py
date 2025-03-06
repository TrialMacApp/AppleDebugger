import argparse
import os
import subprocess
import plistlib
import shutil


def check_app_structure(app_path: str):
    """查看 app 目录是否满足最基本的结构"""
    assert os.path.exists(app_path), f"App path does not exist: {app_path}"
    info_plist_path = os.path.join(app_path, "Info.plist")
    assert os.path.exists(info_plist_path), "App structure error: Missing Info.plist"

    try:
        with open(info_plist_path, "rb") as f:
            plist_data = plistlib.load(f)
    except Exception as e:
        raise ValueError(f"Failed to read Info.plist: {e}")

    executable_file = plist_data.get("CFBundleExecutable")
    assert executable_file, "Info.plist missing CFBundleExecutable key"

    executable_path = os.path.join(app_path, executable_file)
    assert os.path.exists(
        executable_path
    ), "App structure error: Missing executable file"


def unzip_ipa(ipa_path: str) -> str:
    """解压 ipa 并提取 .app 目录"""
    assert os.path.exists(ipa_path), f"The ipa file was not found: {ipa_path}"
    assert os.path.isdir(
        ipa_path
    ), "Invalid ipa format: Expected a file, but got a directory"

    target_directory = os.path.dirname(ipa_path)

    subprocess.run(
        ["/usr/bin/unzip", "-o", ipa_path, "-d", target_directory], check=True
    )

    payload_path = os.path.join(target_directory, "Payload")
    assert os.path.exists(
        payload_path
    ), "Invalid ipa structure: Payload folder is missing"

    app_files = [f for f in os.listdir(payload_path) if f.endswith(".app")]
    if not app_files:
        raise ValueError("Failed to process ipa: No .app found in Payload")

    app_path = os.path.join(payload_path, app_files[0])
    check_app_structure(app_path)

    # 移动 .app 到目标目录，并删除 Payload 文件夹
    shutil.move(app_path, target_directory)
    shutil.rmtree(payload_path)

    return app_files[0]


def check_target_app_dir(targetapp_path: str, del_ipa: bool) -> str:
    assert (
        os.path.basename(targetapp_path) == "TargetApp"
    ), "Wrong directory: The directory name must be 'TargetApp'"
    assert os.path.exists(targetapp_path), "TargetApp directory does not exist"

    app_files = [f for f in os.listdir(targetapp_path) if f.endswith(".app")]
    ipa_files = [f for f in os.listdir(targetapp_path) if f.endswith(".ipa")]

    if not app_files and not ipa_files:
        raise FileNotFoundError(
            "The executable file is missing, you should drag and drop the ipa or app into the TargetApp directory"
        )

    if len(app_files) > 1:
        raise ValueError("Multiple .app files found, please delete redundant files")

    if len(ipa_files) > 1:
        raise ValueError("Multiple .ipa files found, please delete redundant files")

    if len(app_files) == 1:
        return app_files[0]

    ipa_file = ipa_files[0]
    app_name = unzip_ipa(ipa_file)

    # 根据 del_ipa 选项删除 ipa 文件
    if del_ipa:
        ipa_path = os.path.join(targetapp_path, ipa_file)
        shutil.rmtree(ipa_path) if os.path.isdir(ipa_path) else os.remove(ipa_path)

    return app_name


def del_useless_files(build_app_path: str):
    for item in os.listdir(build_app_path):
        item_path = os.path.join(build_app_path, item)
        if os.path.isdir(item_path) and (
            item.endswith(".app") or item.endswith(".app.dSYM")
        ):
            try:
                subprocess.run(["rm", "-r", item_path], check=True)
            except Exception as e:
                print(f"Warin: del_useless_files error - {e}")


def check_frameworks(BUILT_PRODUCTS_DIR: str, TARGET_NAME: str):
    dylib_name = f"lib{TARGET_NAME}Dylib.dylib"
    app_path = os.path.join(BUILT_PRODUCTS_DIR, f"{TARGET_NAME}.app")
    frameworks_path = os.path.join(app_path, "Frameworks")
    dylib_dest = os.path.join(frameworks_path, dylib_name)
    dylib_src = os.path.join(BUILT_PRODUCTS_DIR, dylib_name)

    # 确保 Frameworks 目录存在
    os.makedirs(frameworks_path, exist_ok=True)

    if not os.path.exists(dylib_dest):
        shutil.copy2(dylib_src, dylib_dest)


def codesign(identity: str, build_app_path: str):
    """使用指定的身份对应用及其 Frameworks 进行签名"""

    plugins_path = os.path.join(build_app_path, "PlugIns")
    if os.path.exists(plugins_path):
        shutil.rmtree(plugins_path)

    try:
        subprocess.run(
            ["/usr/bin/codesign", "-f", "-s", identity, build_app_path],
            check=True,
        )
    except subprocess.CalledProcessError as e:
        print(f"Error: Failed to sign {build_app_path} - {e}")
        return

    # 获取 Frameworks 目录并签名
    frameworks_path = os.path.join(build_app_path, "Frameworks")
    if os.path.exists(frameworks_path):
        for framework in os.listdir(frameworks_path):
            framework_path = os.path.join(frameworks_path, framework)
            try:
                subprocess.run(
                    [
                        "/usr/bin/codesign",
                        "-f",
                        "-s",
                        identity,
                        "--deep",
                        framework_path,
                    ],
                    check=True,
                )
            except subprocess.CalledProcessError as e:
                print(f"Warning: Failed to sign {framework_path} - {e}")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="AppleDebugger Tool")

    required_group = parser.add_argument_group("Required")
    required_group.add_argument(
        "-m", "--mode", choices=["lite", "general", "pro"], required=True, help=""
    )

    parser.add_argument("--codesign", action="store_true", help="")
    parser.add_argument(
        "--del-useless-files",
        action="store_true",
        help="Delete TargetApp and dSYM that are repeatedly packaged into the product",
    )
    parser.add_argument(
        "--del-ipa",
        action="store_true",
        help="Delete the ipa file and it will be automatically decompressed into an app",
    )

    subparsers = parser.add_subparsers(dest="command", help="")
    parser_add = subparsers.add_parser("info", help="Modify info.plist")
    parser_add.add_argument(
        "-modify-display-name", type=str, help="Modify the display name"
    )
    parser_add.add_argument("-modify-version", type=str, help="Modify the version")
    parser_add.add_argument(
        "-modify-short-version", type=str, help="Modify the short version"
    )
    parser_add.add_argument(
        "--del-supported-devices", action="store_true", help="Remove model restrictions"
    )

    args = parser.parse_args()

    print("*" * 10 + " AppleDebugger " + "*" * 10)

    # 读取环境变量
    BUILT_PRODUCTS_DIR = os.environ.get("BUILT_PRODUCTS_DIR", "")
    TARGET_NAME = os.environ.get("TARGET_NAME", "")
    SRCROOT = os.environ.get("SRCROOT", "")
    PRODUCT_BUNDLE_IDENTIFIER = os.environ.get("PRODUCT_BUNDLE_IDENTIFIER", "")
    EXPANDED_CODE_SIGN_IDENTITY = os.environ.get("EXPANDED_CODE_SIGN_IDENTITY", "")
    TargetApp_path = os.path.join(SRCROOT, TARGET_NAME, "TargetApp")
    build_app_path = os.path.join(BUILT_PRODUCTS_DIR, TARGET_NAME + ".app")
    build_app_plist_path = os.path.join(build_app_path, "Info.plist")

    # print(args._get_args())
    # print(args._get_kwargs())
    # exit()

    if args.del_useless_files:
        del_useless_files(build_app_path)
        exit(0)

    if args.codesign:
        check_frameworks(BUILT_PRODUCTS_DIR, TARGET_NAME)
        codesign(EXPANDED_CODE_SIGN_IDENTITY, build_app_path)
        exit(0)

    target_app_name = check_target_app_dir(TargetApp_path, args.del_ipa)
    target_app_path = os.path.join(TargetApp_path, target_app_name)

    shutil.rmtree(build_app_path, ignore_errors=True)
    shutil.copytree(target_app_path, build_app_path)

    result = subprocess.run(
        [
            "/usr/libexec/PlistBuddy",
            "-c",
            "Print CFBundleExecutable",
            build_app_plist_path,
        ],
        check=True,
        text=True,
        capture_output=True,
    )
    executable_file_name = result.stdout.strip()

    subprocess.run(
        [
            "/usr/libexec/PlistBuddy",
            "-c",
            "Set :CFBundleIdentifier " + PRODUCT_BUNDLE_IDENTIFIER,
            build_app_plist_path,
        ],
        check=True,
    )

    if args.mode != "lite":
        subprocess.run(
            [
                "/opt/AppleDebugger/bin/optool",
                "install",
                "-p",
                "@rpath/lib" + TARGET_NAME + "Dylib.dylib",
                "-t",
                os.path.join(build_app_path, executable_file_name),
            ],
            check=True,
        )
