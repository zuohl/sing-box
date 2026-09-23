#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

echo "=== sing-box Android AAR 构建流程 ==="
echo "工作目录: ${REPO_DIR}"

# 1. 检查 Go
if ! command -v go &>/dev/null; then
  echo "错误: 未找到 Go 工具链，请安装 Go (>=1.24)。" >&2
  exit 1
fi
echo "Go 版本: $(go version)"

# 2. 检查 Java
if ! command -v java &>/dev/null && [ -z "${JAVA_HOME:-}" ]; then
  echo "错误: 未找到 Java，请配置 JAVA_HOME 或将 java 加入 PATH。" >&2
  exit 1
fi
echo "Java: $(java -version 2>&1 | head -n 1)"

# 3. 检查 Android SDK / NDK 环境变量
if [ -z "${ANDROID_HOME:-}" ]; then
  if [ -n "${ANDROID_SDK_ROOT:-}" ] && [ -d "${ANDROID_SDK_ROOT}" ]; then
    export ANDROID_HOME="${ANDROID_SDK_ROOT}"
  elif [ -d "/usr/local/lib/android/sdk" ]; then
    export ANDROID_HOME="/usr/local/lib/android/sdk"
  elif [ -d "${HOME}/Library/Android/sdk" ]; then
    export ANDROID_HOME="${HOME}/Library/Android/sdk"
  elif [ -d "${HOME}/Android/Sdk" ]; then
    export ANDROID_HOME="${HOME}/Android/Sdk"
  fi
fi

if [ -n "${ANDROID_HOME:-}" ]; then
  echo "ANDROID_HOME: ${ANDROID_HOME}"
else
  echo "警告: 未设置 ANDROID_HOME，构建可能会失败。" >&2
fi

if [ -n "${ANDROID_NDK_HOME:-}" ]; then
  echo "ANDROID_NDK_HOME: ${ANDROID_NDK_HOME}"
fi

# 4. 检查/安装 gomobile 和 gobind
GOPATH_BIN="$(go env GOPATH)/bin"
export PATH="${GOPATH_BIN}:${PATH}"

if ! command -v gomobile &>/dev/null || ! command -v gobind &>/dev/null; then
  echo "正在安装 gomobile / gobind (v0.1.13)..."
  go install -v github.com/sagernet/gomobile/cmd/gomobile@v0.1.13
  go install -v github.com/sagernet/gomobile/cmd/gobind@v0.1.13
fi

echo "gomobile: $(command -v gomobile)"
echo "gobind: $(command -v gobind)"

# 5. 执行 build_libbox
cd "${REPO_DIR}"
export SKIP_JAVA_CHECK="${SKIP_JAVA_CHECK:-1}"

echo "开始编译 libbox.aar (参数: $@)..."
go run ./cmd/internal/build_libbox -target android "$@"

echo "=== 构建完成 ==="
if [ -f "${REPO_DIR}/libbox.aar" ]; then
  ls -lh "${REPO_DIR}/libbox.aar"
fi
if [ -f "${REPO_DIR}/libbox-legacy.aar" ]; then
  ls -lh "${REPO_DIR}/libbox-legacy.aar"
fi
