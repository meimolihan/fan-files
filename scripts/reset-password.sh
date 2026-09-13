#!/usr/bin/env bash
set -euo pipefail

BIN="/usr/local/bin/fan-files"
CONFIG="/etc/fan-files/settings.json"

usage() {
  echo "用法: $(basename "$0") <用户名> <新密码>"
  echo "示例: $(basename "$0") admin '新密码'"
}

if [ "$#" -ne 2 ]; then
  usage
  exit 1
fi

username="$1"
password="$2"

if [ -z "$username" ]; then
  echo "错误: 用户名不能为空" >&2
  exit 1
fi

if [ -z "$password" ]; then
  echo "错误: 新密码不能为空" >&2
  exit 1
fi

if [ ! -x "$BIN" ]; then
  echo "错误: 未找到可执行的 $BIN" >&2
  exit 1
fi

if [ ! -r "$CONFIG" ]; then
  echo "错误: 未找到配置文件 $CONFIG" >&2
  exit 1
fi

if [ "$EUID" -ne 0 ]; then
  echo "提示: 非 root，自动用 sudo 重试..."
  exec sudo "$(realpath "$BASH_SOURCE")" "$@"
fi

"$BIN" -c "$CONFIG" users update "$username" --password="$password"

echo "完成: 已重置用户 '$username' 的密码。"