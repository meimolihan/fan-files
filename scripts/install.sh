#!/usr/bin/env bash
#
# fan-files - 文件管理器安装脚本
# 将构建产物（bin/fan-files 或仓库根目录 ./fan-files）安装为 systemd 服务。
# 可重复执行，升级等同于重新安装（覆盖二进制并按需重建配置并重启服务）。
#
# Usage:
#   交互式安装（将提示端口、数据目录与存储根目录）:
#     bash scripts/install.sh
#   参数静默安装（-p 端口 / -d 数据目录 / -r 主存储根 / --extra-root 附加存储 / -b 二进制源）:
#     bash scripts/install.sh -p 8678 -d /var/lib/fan-files -r /vol1/1000 --extra-root "存储空间2=/vol2/1000"
#     bash scripts/install.sh -y

set -euo pipefail

# ================== terminal colors ==================
list_color_init() {
    export gl_hui=$'\033[38;5;59m'
    export gl_hong=$'\033[38;5;9m'
    export gl_lv=$'\033[38;5;10m'
    export gl_huang=$'\033[38;5;11m'
    export gl_lan=$'\033[38;5;32m'
    export gl_bai=$'\033[38;5;15m'
    export gl_zi=$'\033[38;5;13m'
    export gl_bufan=$'\033[38;5;14m'
    export reset=$'\033[0m'
}
list_color_init

sep_line() {
  printf '%s' "$gl_bufan"
  printf '—%.0s' {1..32}
  printf '%s\n' "$reset"
}

section() {
  printf "  %s %s\n" "${gl_zi}▶${reset}" "$1"
}

ok() {
  printf "  %s %s\n" "${gl_lv}>>>${reset}" "$1"
}

skip() {
  printf "  %s %s\n" "${gl_hui}--${reset}" "$1"
}

print_banner() {
  local z="$gl_zi" r="$reset" b="$gl_bai" l="$gl_lan"
  printf '%s\n' \
    "" \
    "  ${z}┌─────────────────────────────────────────┐${r}" \
    "  ${z}│${r}   ${b}fan-files${r}  ${l}文件管理器 · 安装${r}     ${z}│${r}" \
    "  ${z}└─────────────────────────────────────────┘${r}" \
    ""
}

error() { printf "  %s %s\n" "${gl_hong}[错误]${reset}" "$1" >&2; exit 1; }

# ================== customize me ==================
APP_NAME="fan-files"
DEFAULT_PORT=8678
DEFAULT_DATA_DIR="/var/lib/${APP_NAME}"
DEFAULT_ROOT="/vol1/1000"
BIN_PATH="/usr/local/bin/${APP_NAME}"
RECORD_FILE="/etc/${APP_NAME}.conf"
SETTINGS_DIR="/etc/${APP_NAME}"
SETTINGS_FILE="${SETTINGS_DIR}/settings.json"
SERVICE_FILE="/etc/systemd/system/${APP_NAME}.service"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-}")" && pwd)"
DEFAULT_BIN_SRC="${SCRIPT_DIR}/../bin/${APP_NAME}"

# 经 curl|bash 远程执行时，SCRIPT_DIR 指向 bash 抽取的临时目录，本地构建产物
# 需按常见目录回退探测（当前目录 / 上一级目录），否则会误判"无本地产物"。
resolve_local_src() {
  local candidates=(
    "${SCRIPT_DIR:-}/../$1"
    "${SCRIPT_DIR:-}/../${APP_NAME}"
    "$(pwd)/$1"
    "$(pwd)/${APP_NAME}"
    "$(pwd)/../$1"
    "$(dirname "$(pwd)")/$1"
  )
  for c in "${candidates[@]}"; do
    if [ -e "${c}" ]; then
      printf '%s' "${c}"
      return 0
    fi
  done
  return 1
}
# ==================================================

PORT=""
DATA_DIR=""
ROOT=""
EXTRA_ROOTS=""
EXTRA_COUNT=0
BIN_SRC=""
BIN_SRC_EXPLICIT=0
INSTALL_YES=0

# ---- bootstrap: support `bash -c "$(curl ...)" -p ... -d ...` ----
case "$0" in
  -*) set -- "$0" "$@" ;;
esac

# ---- parse command-line args (silent install) ----
while [ "$#" -gt 0 ]; do
  case "$1" in
    -p|--port)
      shift
      [ -n "${1:-}" ] || error "缺少 -p/--port 的值"
      PORT="$1"
      ;;
    -d|--data)
      shift
      [ -n "${1:-}" ] || error "缺少 -d/--data 的值"
      DATA_DIR="$1"
      ;;
    -r|--root)
      shift
      [ -n "${1:-}" ] || error "缺少 -r/--root 的值"
      ROOT="$1"
      ;;
    --extra-root)
      shift
      [ -n "${1:-}" ] || error "缺少 --extra-root 的值（格式: 标签=绝对路径 或 绝对路径）"
      label=""
      pth="$1"
      if [[ "$pth" == *=* ]]; then label="${pth%%=*}"; pth="${pth#*=}"; fi
      [[ "$pth" = /* ]] || error "--extra-root 路径必须是绝对路径: ${pth}"
      EXTRA_ROOTS="${EXTRA_ROOTS} ${label}|${pth}"
      EXTRA_COUNT=$((EXTRA_COUNT + 1))
      ;;
    -b|--bin)
      shift
      [ -n "${1:-}" ] || error "缺少 -b/--bin 的值"
      BIN_SRC="$1"
      BIN_SRC_EXPLICIT=1
      ;;
    -y|--yes)
      INSTALL_YES=1
      ;;
    -h|--help)
      printf "%s\n" "${gl_lan}fan-files${reset} - ${gl_bai}文件管理器 安装脚本${reset}"
      printf "  %-13s %s\n" "${gl_bai}用法:${reset}" "bash scripts/install.sh [-p PORT] [-d DATA_DIR] [-r ROOT] [--extra-root LABEL=PATH] [-b BIN] [-y]"
      printf "  %-13s %s\n" "${gl_bai}-p, --port${reset}" "监听端口（默认 ${gl_lan}${DEFAULT_PORT}${reset}）"
      printf "  %-13s %s\n" "${gl_bai}-d, --data${reset}" "数据目录（默认 ${gl_lan}${DEFAULT_DATA_DIR}${reset}）"
      printf "  %-13s %s\n" "${gl_bai}-r, --root${reset}" "主存储根目录（默认 ${gl_lan}${DEFAULT_ROOT}${reset}）"
      printf "  %-13s %s\n" "${gl_bai}--extra-root${reset}" "附加存储根，可重复指定，格式 ${gl_lan}标签=绝对路径${reset}（留空标签则自动编号）"
      printf "  %-13s %s\n" "${gl_bai}-b, --bin${reset}" "二进制源路径（默认 ${gl_lan}${DEFAULT_BIN_SRC}${reset}）"
      printf "  %-13s %s\n" "${gl_bai}-y, --yes${reset}" "免交互，未指定项全部使用默认值"
      printf "  %-13s %s\n" "${gl_bai}-h, --help${reset}" "显示本帮助"
      printf "%s\n" "${gl_hui}指定任意参数即进入静默安装；不带参数则为交互式安装。${reset}"
      printf "%s\n" "${gl_hui}未指定 -b 且本地无构建产物时，自动从 GitHub Release 下载对应架构二进制。${reset}"
      exit 0
      ;;
    *)
      error "未知参数: $1（使用 -h 查看帮助）"
      ;;
  esac
  shift
done

# ---- read previous install record to prefill defaults (reinstall/upgrade) ----
read_record() {
  [ -f "${RECORD_FILE}" ] || return 0
  while IFS='=' read -r KEY VALUE; do
    KEY=$(printf '%s' "$KEY" | tr -d ' ')
    VALUE=$(printf '%s' "$VALUE" | tr -d '\r')
    case "$KEY" in
      BIN_PATH) [ -n "$VALUE" ] && BIN_PATH="$VALUE" ;;
      PORT) [ -n "$VALUE" ] && [ -z "$PORT" ] && PORT="$VALUE" ;;
      DATA_DIR) [ -n "$VALUE" ] && [ -z "$DATA_DIR" ] && DATA_DIR="$VALUE" ;;
      ROOT) [ -n "$VALUE" ] && [ -z "$ROOT" ] && ROOT="$VALUE" ;;
      EXTRA_ROOTS) [ -n "$VALUE" ] && [ -z "$EXTRA_ROOTS" ] && EXTRA_ROOTS="$VALUE" ;;
    esac
  done < "${RECORD_FILE}"
  return 0
}

# ---- firewall: automatically open the listen port ----
FW_OPENED="n"
open_firewall_port() {
  local PORT="$1"
  if command -v firewall-cmd >/dev/null 2>&1 && firewall-cmd --state >/dev/null 2>&1; then
    if ! firewall-cmd --query-port="${PORT}/tcp" >/dev/null 2>&1; then
      firewall-cmd --permanent --add-port="${PORT}/tcp" >/dev/null 2>&1 || true
      firewall-cmd --reload >/dev/null 2>&1 || true
    fi
    ok "已通过 ${gl_bai}firewalld${reset} 开放端口 ${gl_lan}${PORT}/tcp${reset}"
    FW_OPENED="y"
    return 0
  fi

  if command -v ufw >/dev/null 2>&1 && ufw status 2>/dev/null | grep -q "Status: active"; then
    if ! ufw status 2>/dev/null | grep -q "${PORT}/tcp"; then
      ufw allow "${PORT}/tcp" >/dev/null 2>&1 || true
    fi
    ok "已通过 ${gl_bai}ufw${reset} 开放端口 ${gl_lan}${PORT}/tcp${reset}"
    FW_OPENED="y"
    return 0
  fi

  if command -v iptables >/dev/null 2>&1; then
    if iptables -C INPUT -p tcp --dport "${PORT}" -j ACCEPT >/dev/null 2>&1; then
      ok "端口 ${gl_lan}${PORT}/tcp${reset} 已在 iptables 中放行"
      FW_OPENED="y"
      return 0
    fi
    if iptables -L INPUT -n 2>/dev/null | grep -qE 'policy (DROP|REJECT)|REJECT|DROP'; then
      if iptables -I INPUT -p tcp --dport "${PORT}" -j ACCEPT >/dev/null 2>&1; then
        ok "已通过 ${gl_bai}iptables${reset} 开放端口 ${gl_lan}${PORT}/tcp${reset}"
        FW_OPENED="y"
        return 0
      fi
    fi
  fi
  printf "  %s %s\n" "${gl_huang}[提示]${reset}" "未检测到活跃的防火墙（firewalld/ufw/iptables），跳过端口开放。"
}

[ "$(id -u)" != "0" ] && error "请以 root 身份运行（例如 sudo bash scripts/install.sh）"

read_record

# 从安装记录恢复附加存储配置时补齐计数
if [ -n "${EXTRA_ROOTS}" ] && [ "${EXTRA_COUNT}" -eq 0 ]; then
  EXTRA_COUNT=$(printf '%s' "${EXTRA_ROOTS}" | wc -w | tr -d ' ')
fi

print_banner
sep_line
section "安装信息"
printf "  %-14s %s\n" "${gl_lan}系统${reset}" "$(uname -s) $(uname -m)"
printf "  %-14s %s\n" "${gl_lan}程序${reset}" "${gl_bai}${APP_NAME}${reset}"
sep_line

# ---- silent install detection ----
SILENT="n"
if [ -n "${PORT}" ]; then
  case "${PORT}" in
    ''|*[!0-9]*) error "PORT 无效（需为 1‑65535 的数字）: ${PORT}" ;;
    *) [ "${PORT}" -ge 1 ] && [ "${PORT}" -le 65535 ] || error "PORT 超出范围（1‑65535）: ${PORT}" ;;
  esac
  SILENT="y"
fi
if [ -n "${DATA_DIR}" ]; then
  SILENT="y"
fi
if [ -n "${ROOT}" ]; then
  [[ "${ROOT}" = /* ]] || error "ROOT 必须是绝对路径: ${ROOT}"
  SILENT="y"
fi
if [ "$EXTRA_COUNT" -gt 0 ]; then
  SILENT="y"
fi
if [ -n "${BIN_SRC}" ]; then
  SILENT="y"
fi
if [ ! -t 0 ]; then
  SILENT="y"
fi

section "配置参数"
# port prompt
if [ -z "${PORT}" ]; then
  if [ "$INSTALL_YES" = "1" ] || [ ! -t 0 ]; then
    PORT="${DEFAULT_PORT}"
  else
    while :; do
      read -r -p "${gl_bai}请输入监听端口${reset} ${gl_hui}[默认: ${DEFAULT_PORT}]${reset}: " PORT
      PORT="${PORT:-$DEFAULT_PORT}"
      case "$PORT" in
        ''|*[!0-9]*) printf "  %s\n" "${gl_huang}端口无效，请重新输入。${reset}" ;;
        *)
          if [ "$PORT" -ge 1 ] && [ "$PORT" -le 65535 ]; then break; fi
          printf "  %s\n" "${gl_huang}端口超出范围（1‑65535），请重新输入。${reset}"
          ;;
      esac
    done
  fi
else
  printf "  %-14s %s\n" "${gl_lan}监听端口${reset}" "${gl_bai}${PORT}${reset}（参数指定）"
fi
PORT="${PORT:-$DEFAULT_PORT}"

# data dir prompt
if [ -z "${DATA_DIR}" ]; then
  if [ "$INSTALL_YES" = "1" ] || [ ! -t 0 ]; then
    DATA_DIR="${DEFAULT_DATA_DIR}"
  else
    read -r -p "${gl_bai}请输入数据目录${reset} ${gl_hui}[默认: ${DEFAULT_DATA_DIR}]${reset}: " DATA_DIR
    DATA_DIR="${DATA_DIR:-$DEFAULT_DATA_DIR}"
  fi
else
  printf "  %-14s %s\n" "${gl_lan}数据目录${reset}" "${gl_bai}${DATA_DIR}${reset}（参数指定）"
fi
DATA_DIR="${DATA_DIR:-$DEFAULT_DATA_DIR}"

# storage root prompt
if [ -z "${ROOT}" ]; then
  if [ "$INSTALL_YES" = "1" ] || [ ! -t 0 ]; then
    ROOT="${DEFAULT_ROOT}"
  else
    read -r -p "${gl_bai}请输入主存储根目录${reset} ${gl_hui}[默认: ${DEFAULT_ROOT}]${reset}: " ROOT
    ROOT="${ROOT:-$DEFAULT_ROOT}"
  fi
else
  printf "  %-14s %s\n" "${gl_lan}主存储根${reset}" "${gl_bai}${ROOT}${reset}（参数指定）"
fi
ROOT="${ROOT:-$DEFAULT_ROOT}"

# extra roots
if [ "$EXTRA_COUNT" -gt 0 ]; then
  printf "  %-14s %s\n" "${gl_lan}附加存储根${reset}" "${gl_bai}${EXTRA_COUNT}${reset}（参数指定）"
elif [ "$INSTALL_YES" = "1" ] || [ ! -t 0 ] || [ -n "${EXTRA_ROOTS}" ]; then
  skip "未配置附加存储根（可使用 --extra-root 标签=/绝对路径 添加）"
else
  read -r -p "${gl_bai}请输入附加存储根（多个以空格分隔，格式 标签=/绝对路径，留空跳过）${reset}: " EXTRA_NEW
  if [ -n "${EXTRA_NEW}" ]; then
    for ep in ${EXTRA_NEW}; do
      lbl=""
      pp="$ep"
      if [[ "${ep}" == *=* ]]; then lbl="${ep%%=*}"; pp="${ep#*=}"; fi
      EXTRA_ROOTS="${EXTRA_ROOTS} ${lbl}|${pp}"
      EXTRA_COUNT=$((EXTRA_COUNT + 1))
    done
  fi
fi

# binary source
BIN_SRC="${BIN_SRC:-$DEFAULT_BIN_SRC}"
if [ ! -f "${BIN_SRC}" ]; then
  # 经 curl|bash 远程执行：探测当前目录/仓库目录中的本地产物
  if [ "${BIN_SRC_EXPLICIT}" != "1" ]; then
    DISCOVERED_BIN="$(resolve_local_src "bin/${APP_NAME}")" || true
    if [ -n "${DISCOVERED_BIN:-}" ]; then
      ok "已从仓库目录发现本地产物 ${gl_bai}${DISCOVERED_BIN}${reset}"
      BIN_SRC="${DISCOVERED_BIN}"
    fi
  fi
fi
if [ ! -f "${BIN_SRC}" ]; then
  if [ "${BIN_SRC_EXPLICIT}" = "1" ]; then
    error "未找到二进制文件 ${BIN_SRC}（-b 显式指定）"
  fi
  # 本地无构建产物时，尝试从 GitHub Release 下载指定架构的静态二进制
  REL_ARCH=""
  case "$(uname -m)" in
    x86_64|amd64) REL_ARCH="amd64" ;;
    aarch64|arm64) REL_ARCH="arm64" ;;
    *) error "不支持的架构: $(uname -m)，请先本地构建（scripts/build-and-push.sh 或直接 go build）或使用 -b 指定" ;;
  esac
  REL_URL="https://github.com/meimolihan/fan-files/releases/latest/download/fan-files_linux_${REL_ARCH}"
  ok "本地无构建产物，尝试从 GitHub Release 下载 ${gl_bai}${REL_URL}${reset}"
  TMP_BIN="$(mktemp)"
  if ! curl -fsSL "${REL_URL}" -o "${TMP_BIN}"; then
    error "下载 Release 二进制失败（${REL_URL}），请先本地构建或使用 -b 指定"
  fi
  chmod +x "${TMP_BIN}"
  BIN_SRC="${TMP_BIN}"
  ok "已从 GitHub Release 下载二进制（${gl_bai}$(du -h "${TMP_BIN}" | cut -f1)${reset}）"
fi

if command -v systemctl >/dev/null 2>&1; then
  USE_SYSTEMD="y"
else
  USE_SYSTEMD="n"
  printf "  %s\n" "${gl_huang}[警告]${reset} 未检测到 systemd（容器或受限环境）。"
  printf "  %s\n" "${gl_hui}    已回退为后台运行模式，重启或崩溃后服务不会自动恢复。${reset}"
fi

sep_line
section "安装程序"
ok "正在安装 ${gl_bai}${APP_NAME}${reset} 二进制 ${gl_hong}.${gl_huang}.${gl_lv}.${gl_bai}"

cp -f "${BIN_SRC}" "${BIN_PATH}"
chmod +x "${BIN_PATH}"
ok "已安装二进制至 ${gl_bai}${BIN_PATH}${reset}"

ok "正在创建数据目录 ${gl_lan}${DATA_DIR}${reset}"
mkdir -p "${DATA_DIR}"
chmod 700 "${DATA_DIR}"

# ---- write install record ----
mkdir -p "$(dirname "${RECORD_FILE}")"
cat > "${RECORD_FILE}" <<EOF
# ${APP_NAME} 安装记录（由 install.sh 生成，请勿手动修改）
BIN_PATH=${BIN_PATH}
PORT=${PORT}
DATA_DIR=${DATA_DIR}
ROOT=${ROOT}
EXTRA_ROOTS=${EXTRA_ROOTS}
EOF
chmod 0644 "${RECORD_FILE}"
ok "已写入安装记录 ${gl_bai}${RECORD_FILE}${reset}"

# ---- generate settings.json ----
mkdir -p "${SETTINGS_DIR}"
{
  printf '{\n'
  printf '  "port": %s,\n' "${PORT}"
  printf '  "baseURL": "",\n'
  printf '  "address": "0.0.0.0",\n'
  printf '  "log": "stdout",\n'
  printf '  "database": "%s",\n' "${DATA_DIR}/${APP_NAME}.db"
  printf '  "root": "%s"' "${ROOT}"
  idx=1
  entry=""
  label=""
  pth=""
  for entry in ${EXTRA_ROOTS}; do
    label="${entry%%|*}"
    pth="${entry#*|}"
    if [ -z "${label}" ]; then
      label="存储空间$((idx + 1))"
    fi
    if [ "${idx}" -eq 1 ]; then
      printf ',\n'
      printf '  "extraRoots": [\n'
    else
      printf ',\n'
    fi
    printf '    {"path": "%s", "label": "%s"}' "${pth}" "${label}"
    idx=$((idx + 1))
  done
  if [ "${idx}" -gt 1 ]; then
    printf '\n  ]\n'
  else
    printf '\n'
  fi
  printf '}\n'
} > "${SETTINGS_FILE}"
chmod 0644 "${SETTINGS_FILE}"
ok "已写入配置文件 ${gl_bai}${SETTINGS_FILE}${reset}"

sep_line
section "启动服务"
if [ "${USE_SYSTEMD}" = "y" ]; then
  cat > "${SERVICE_FILE}" <<UNIT
[Unit]
Description=${APP_NAME} - 文件管理器
After=network-online.target local-fs.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=${BIN_PATH} --config=${SETTINGS_FILE}
WorkingDirectory=${DATA_DIR}
Environment=TZ=Asia/Shanghai
Restart=on-failure
RestartSec=3

[Install]
WantedBy=multi-user.target
UNIT

  systemctl daemon-reload
  systemctl enable "${APP_NAME}" >/dev/null 2>&1 || true
  systemctl restart "${APP_NAME}"
  sleep 2
  if systemctl is-active "${APP_NAME}" >/dev/null 2>&1; then
    ok "${gl_bai}${APP_NAME}${reset} 服务已启动。"
    systemctl status "${APP_NAME}" --no-pager || true
  else
    printf "  %s\n" "${gl_hong}[错误]${reset} 服务启动失败，请检查：${gl_bai}journalctl -u ${APP_NAME} -n 50${reset}" >&2
    exit 1
  fi
else
  if command -v pgrep >/dev/null 2>&1 && pgrep -x "${APP_NAME}" >/dev/null 2>&1; then
    printf "  %s\n" "${gl_huang}[警告]${reset} 检测到 ${APP_NAME} 进程可能已在运行"
  else
    nohup "${BIN_PATH}" --config="${SETTINGS_FILE}" >> "${DATA_DIR}/${APP_NAME}.log" 2>&1 &
    ok "${APP_NAME} 已在后台启动，pid: ${gl_bai}$!${reset}"
  fi
fi

# 取第一个IPv4
IP=$(hostname -I 2>/dev/null | awk '{print $1}')
[ -z "${IP}" ] && IP="<服务器IP>"

open_firewall_port "${PORT}"

if [ "${FW_OPENED}" = "y" ]; then
  FW_STATUS="${gl_lv}已开放 ${PORT}/tcp${reset}"
else
  FW_STATUS="${gl_huang}未检测到活跃防火墙，已跳过${reset}"
fi

sep_line
if [ "${USE_SYSTEMD}" = "y" ]; then
  printf "  %s\n" "${gl_lv}✔ ${APP_NAME} 安装成功！${reset}"
  printf "  %-14s %s\n" "${gl_lan}访问地址${reset}" "${gl_bai}http://${IP}:${PORT}${reset}"
  printf "  %-14s %s\n" "${gl_lan}数据目录${reset}" "${gl_bai}${DATA_DIR}${reset}"
  printf "  %-14s %s\n" "${gl_lan}配置文件${reset}" "${gl_bai}${SETTINGS_FILE}${reset}"
  printf "  %-14s %s\n" "${gl_lan}主存储根${reset}" "${gl_bai}${ROOT}${reset}"
  if [ "${EXTRA_COUNT}" -gt 0 ]; then
    printf "  %-14s %s\n" "${gl_lan}附加存储根${reset}" "${gl_bai}${EXTRA_COUNT} 个${reset}"
  fi
  printf "  %-14s %s\n" "${gl_lan}防火墙状态${reset}" "$FW_STATUS"
  printf "  %-14s %s\n" "${gl_lan}运行模式${reset}" "${gl_bai}systemd 服务${reset}"
  sep_line
  printf "  %s\n" "${gl_bai}常用命令：${reset}"
  printf "    %-46s %s\n" "${gl_hui}systemctl status ${APP_NAME}${reset}" "${gl_lan}# 查看状态${reset}"
  printf "    %-46s %s\n" "${gl_hui}systemctl restart ${APP_NAME}${reset}" "${gl_lan}# 重启服务${reset}"
  printf "    %-46s %s\n" "${gl_hui}systemctl stop ${APP_NAME}${reset}" "${gl_lan}# 停止服务${reset}"
  printf "    %-46s %s\n" "${gl_hui}journalctl -u ${APP_NAME} -f${reset}" "${gl_lan}# 跟随日志${reset}"
  printf "    %-46s %s\n" "${gl_hui}journalctl -u ${APP_NAME} -n 50${reset}" "${gl_lan}# 最近日志${reset}"
else
  printf "  %s\n" "${gl_lv}✔ ${APP_NAME} 安装成功！${reset} ${gl_huang}（后台运行模式）${reset}"
  printf "  %-14s %s\n" "${gl_lan}访问地址${reset}" "${gl_bai}http://${IP}:${PORT}${reset}"
  printf "  %-14s %s\n" "${gl_lan}数据目录${reset}" "${gl_bai}${DATA_DIR}${reset}"
  printf "  %-14s %s\n" "${gl_lan}配置文件${reset}" "${gl_bai}${SETTINGS_FILE}${reset}"
  printf "  %s\n" "  ${gl_huang}注意：${reset}后台运行模式在系统重启后不会自动恢复。"
fi

sep_line