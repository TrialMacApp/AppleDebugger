#!/bin/bash

# 定义颜色
BLUE='\033[0;34m'
GREEN='\033[0;32m'
NC='\033[0m'

# 定义目标目录
XCODE_TEMPLATES_DIR="$HOME/Library/Developer/Xcode/Templates"

# 创建环境变量配置文件
BASH_PROFILE="$HOME/.bash_profile"
ZSH_PROFILE="$HOME/.zshrc"

echo -e "${BLUE}Start installing Xcode custom templates.${NC}"

# 创建目标目录
mkdir -p "$XCODE_TEMPLATES_DIR"

# 连接模板
ln -fhs "$AppleDebuggerPath/templates" "$XCODE_TEMPLATES_DIR/AppleDebugger"

# 添加环境变量
echo "# AppleDebugger " >> "$BASH_PROFILE"
echo "export AppleDebuggerPath=/opt/AppleDebugger" >> "$BASH_PROFILE"

# 一般来说必然用zsh
if [ -f "$ZSH_PROFILE" ]; then
    echo "# AppleDebugger " >> "$ZSH_PROFILE"
    echo "export AppleDebuggerPath=/opt/AppleDebugger" >> "$ZSH_PROFILE"
fi

echo -e "${GREEN}Installation completed !${NC}"
echo -e "${GREEN}Restart Xcode to use the new templates.${NC}"
