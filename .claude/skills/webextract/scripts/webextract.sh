#!/usr/bin/env bash
# webextract 包装脚本
# 作用：随 skill 分发预编译二进制，让克隆项目的人无需安装即可使用。
# 行为：
#   1. 自动检测当前 OS/架构，匹配同目录下 webextract-<os>-<arch> 二进制；
#   2. 修复下载/解压后可能丢失的可执行权限（git clone 通常保留，zip 下载会丢）；
#   3. 找不到匹配二进制时，回退到从项目源码 go build；
#   4. 透传所有参数给真正的 webextract。
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ---------- 归一化平台标识 ----------
OS="$(uname -s)"
ARCH="$(uname -m)"
case "$OS" in
  Darwin)                       OS="darwin" ;;
  Linux)                        OS="linux" ;;
  MINGW*|MSYS*|CYGWIN*)         OS="windows" ;;
esac
case "$ARCH" in
  x86_64|amd64)                 ARCH="amd64" ;;
  aarch64|arm64)                ARCH="arm64" ;;
esac

PLATFORM_BIN="$SCRIPT_DIR/webextract-${OS}-${ARCH}"

# ---------- 定位并执行二进制 ----------
run_bin() {
  local bin="$1"; shift   # 剥离 bin 本身，剩下才是要透传给 webextract 的参数
  [[ -x "$bin" ]] || chmod +x "$bin" 2>/dev/null || true
  exec "$bin" "$@"
}

if [[ -f "$PLATFORM_BIN" ]]; then
  run_bin "$PLATFORM_BIN" "$@"
fi

# ---------- 回退：从源码构建 ----------
# scripts/ 位于 .claude/skills/webextract/scripts/，项目根的源码在 ../../../../webextract
SRC_DIR="$(cd "$SCRIPT_DIR/../../../../webextract" 2>/dev/null && pwd || true)"
GENERIC_BIN="$SCRIPT_DIR/webextract"

if [[ -n "$SRC_DIR" && -d "$SRC_DIR" ]] && command -v go >/dev/null 2>&1; then
  echo "[webextract.sh] 未找到 $OS/$ARCH 预编译二进制，从源码构建..." >&2
  ( cd "$SRC_DIR" && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "$GENERIC_BIN" ./cmd/webextract ) >&2
  run_bin "$GENERIC_BIN" "$@"
fi

cat >&2 <<EOF
[webextract.sh] 错误：找不到 $OS/$ARCH 的 webextract 二进制。

解决方案（任选其一）：
  1. 安装 Go 1.26+ 后在此项目根目录运行：
       cd webextract && go build -o ../.claude/skills/webextract/scripts/webextract-${OS}-${ARCH} ./cmd/webextract
  2. 重新从仓库克隆，确认 scripts/ 下的二进制完整。

已编译支持的平台：darwin/arm64、darwin/amd64、linux/amd64、linux/arm64。
EOF
exit 127
