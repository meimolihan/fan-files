#!/bin/bash
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

command -v go >/dev/null 2>&1 || error "未找到 go，请先安装并加入 PATH（如 export PATH=\$PATH:/usr/local/go/bin）"
command -v pnpm >/dev/null 2>&1 || error "未找到 pnpm，请先安装 corepack/pnpm"

# ===================== 重复Tag/Release自动清理 =====================
info "检查远端是否存在 Release ${TAG}"
if gh release view "${TAG}" >/dev/null 2>&1; then
    warn "发现已存在Release ${TAG}，准备删除Release并清理tag"
    gh release delete "${TAG}" -y --cleanup-tag
fi

info "清理本地&远端Git tag: ${TAG}"
git tag -d "${TAG}" 2>/dev/null || true
git push origin --delete "${TAG}" 2>/dev/null || true

# ===================== 版本号 bump（内联） =====================
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

# ===================== 构建 =====================
info "构建前端生产包"
(cd frontend && pnpm install --frozen-lockfile && CI=true pnpm run build)

COMMIT_SHA=$(git rev-parse --short HEAD)
info "构建后端 fan-files v${TARGET_VER} (commit ${COMMIT_SHA})"
mkdir -p bin
CGO_ENABLED=0 go build \
    -ldflags="-s -w -X \"github.com/meimolihan/fan-files/version.Version=${TARGET_VER}\" -X \"github.com/meimolihan/fan-files/version.CommitSHA=${COMMIT_SHA}\"" \
    -o bin/fan-files .

info "生成Release资产 bin/fan-files_linux_amd64"
cp ./bin/fan-files ./bin/fan-files_linux_amd64
./bin/fan-files version

# ===================== Git 提交 & Tag =====================
info "提交版本变更"
git add version/version.go frontend/package.json frontend/pnpm-lock.yaml
git commit -m "chore: bump version to ${TARGET_VER}" || info "无版本文件变更，跳过提交"
git push origin main

git tag "${TAG}"
git push origin "${TAG}"

# ===================== GitHub Release =====================
if command -v gh >/dev/null 2>&1; then
    info "检测到 gh cli，准备处理 GitHub Release ${TAG}"
    ans="n"
    if [[ ${YES_MODE} -eq 1 ]]; then
        ans="y"
    else
        read -p "确认创建Release ${TAG} ? [y/N] " ans
    fi

    if [[ "${ans}" =~ ^[yY]$ ]]; then
        info "新建 Release ${TAG}"
        gh release create "${TAG}" ./bin/fan-files_linux_amd64 \
            --title "Release ${TAG}" \
            --generate-notes
        info "✅ GitHub Release 处理完成"
    else
        warn "跳过Release创建"
    fi
else
    warn "未找到 gh cli：仅推送git tag，不会生成网页端GitHub Release"
    warn "安装：apt install gh && gh auth login"
fi

info "✅ 发布流程全部完成 ${TAG}"