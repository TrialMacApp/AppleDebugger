#!/bin/bash
ARCH=$(uname -m)
if [ "$ARCH" == "arm64" ]; then
    EXEC_FILE_PATH="/opt/AppleDebugger/bin/preprocessor_arm"
else
    EXEC_FILE_PATH="/opt/AppleDebugger/bin/preprocessor_x86"
fi
$EXEC_FILE_PATH "$@"