#!/bin/bash
#
# fan-files - 发布脚本（触发 GitHub Actions 自动构建）
# 不在本地编译任何产物：仅更新版本号、推送代码并打 v 开头 tag。
# 推送 tag 后由 GitHub Actions 自动完成全部编译与发布：
#   release.yml -> amd64/arm64 二进制并创建 GitHub Release（版本跟随 version/version.go）
#   build.yml   -> multi-arch Docker 镜像（latest + 版本标签）
#
# Usage:
#   TAG(必填) 形如 v1.0.0; --yes 免交互
#     bash scripts/build-and-push.sh v1.0.0 --yes
set -euo pipefail

info() { echo -e "\033[32m>>> $*\033[0m"; }
warn() { echo -e "\033[33m!!! $*\033[0m"; }
error() { echo -e "\033[31mERROR: $*\033[0m"; exit 1; }

YES_MODE=0
TAG=""
while [[ $# -gt 0 ]]; do
    case "$1" in
        --yes) YES_MODE=1; shift ;;
        *) TAG="$1"; shift ;;
    esac
done

[[ -z "${TAG}" ]] && error "缺少TAG参数，示例: $0 v1.0.0 --yes"

cd "$(dirname "$0")/.."
TARGET_VER="${TAG#v}"

# ===================== 重复Tag/Release自动清理 =====================
info "检查远端是否存在 Release ${TAG}"
if command -v gh >/dev/null 2>&1 && gh release view "${TAG}" >/dev/null 2>&1; then
    warn "发现已存在Release ${TAG}，准备删除Release并清理tag"
    gh release delete "${TAG}" -y --cleanup-tag
fi

info "清理本地&远端Git tag: ${TAG}"
git tag -d "${TAG}" 2>/dev/null || true
git push origin --delete "${TAG}" 2>/dev/null || true

# ===================== 版本号 bump =====================
info "执行版本号更新 ${TARGET_VER}"

sed_i_arg() {
    if sed --version 2>&1 | grep -q GNU; then echo ""
    elif sed --version 2>&1 | grep -q busybox; then echo ""
    else echo "''"
    fi
}

SED_I=$(sed_i_arg)
BUMP_FILES=("version/version.go" "frontend/package.json")
for f in "${BUMP_FILES[@]}"; do
    [[ ! -f "${f}" ]] && error "缺失文件 ${f}"
done

if [[ "${SED_I}" == "''" ]]; then
    sed -i '' 's/^\(\s*Version = \)"[^"]*"/\1"'"${TARGET_VER}"'"/' version/version.go
    sed -i '' "s/^  \"version\": \".*\",\$/  \"version\": \"${TARGET_VER}\",/" frontend/package.json
else
    sed -i 's/^\(\s*Version = \)"[^"]*"/\1"'"${TARGET_VER}"'"/' version/version.go
    sed -i "s/^  \"version\": \".*\",\$/  \"version\": \"${TARGET_VER}\",/" frontend/package.json
fi

info "版本号确认:"
grep -n 'Version = ' version/version.go
grep -n '"version"' frontend/package.json

# ===================== Git 提交 & Tag =====================
info "提交版本变更"
git add version/version.go frontend/package.json frontend/pnpm-lock.yaml
git commit -m "chore: bump version to ${TARGET_VER}" || info "无版本文件变更，跳过提交"
git push origin main

git tag "${TAG}"
git push origin "${TAG}"

# ===================== 交由 CI 自动构建发布 =====================
info "✅ 已推送 tag ${TAG}，GitHub Actions 将自动完成编译与 Release 创建"

info "查看发布结果: gh release view ${TAG}"
info "查看镜像: docker pull mobufan/fan-files:${TAG}"