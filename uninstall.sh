#!/bin/bash

BLUE='\033[0;34m'
GREEN='\033[0;32m'
NC='\033[0m'

# xcode模板目录
XCODE_TEMPLATES_DIR="$HOME/Library/Developer/Xcode/Templates"

echo -e "${BLUE}Start uninstalling Xcode custom templates.${NC}"

# 删除模板文件
rm "$XCODE_TEMPLATES_DIR/AppleDebugger"

# 删除环境变量配置
sed -i '' '/# AppleDebugger /d' "$HOME/.bash_profile"
sed -i '' '/export AppleDebuggerPath=/d' "$HOME/.bash_profile"

if [ -f "$HOME/.zshrc" ]; then
    sed -i '' '/# AppleDebugger /d' "$HOME/.zshrc"
    sed -i '' '/export AppleDebuggerPath=/d' "$HOME/.zshrc"
fi

echo -e "${GREEN}Uninstallation completed!${NC}"