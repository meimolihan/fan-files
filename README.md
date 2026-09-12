# fan-files

轻量级文件管理器，基于 filebrowser 二次开发。支持**多根存储合并**、Web 界面上传/下载/预览/编辑，部署即用。

## 特性
- **多根合并**：主存储根 + 多个附加存储根，首页合并列表，`storage` 字段区分来源
- **原生 systemd**：一键安装/卸载/升级，开机自启，日志统一管理
- **零依赖静态二进制**：单文件部署，无运行时依赖
- **远程一键安装**：`curl | bash` 无需本地构建，自动从 Release 下载对应架构二进制
- **保留数据升级**：卸载保留 `/var/lib/fan-files`，重装即可恢复用户/设置/密码

## 快速开始（远程安装）

```bash
# 静默安装（指定端口、数据目录、主存储、附加存储）
curl -fsSL https://raw.githubusercontent.com/meimolihan/fan-files/main/scripts/install.sh \
  | bash -s -- -y -p 8678 -d /var/lib/fan-files -r /vol1/1000 \
      --extra-root "存储空间2=/vol2/1000"

# 交互式安装（会提示端口、数据目录、存储根）
curl -fsSL https://raw.githubusercontent.com/meimolihan/fan-files/main/scripts/install.sh | bash
```

> **前置条件**：脚本已推送至 `main` 分支，且已发布 Release（含 `fan-files_linux_amd64` 产物）。  
> 首次部署需执行：`./scripts/build-and-push.sh v1.0.0 --yes`（构建、打 tag、创建 Release）。

## 本地构建与开发

```bash
# 前端
cd frontend && pnpm install && CI=true pnpm run build

# 后端（需 Go 1.21+）
export PATH=$PATH:/usr/local/go/bin
CGO_ENABLED=0 go build \
  -ldflags="-s -w -X 'github.com/meimolihan/fan-files/version.Version=1.0.0' \
  -X 'github.com/meimolihan/fan-files/version.CommitSHA=$(git rev-parse HEAD)'" \
  -o fan-files .
```

## systemd 服务管理

安装脚本会自动创建并启用 `fan-files.service`，常用命令：

| 操作 | 命令 |
|------|------|
| 查看状态 | `systemctl status fan-files` |
| 重启服务 | `systemctl restart fan-files` |
| 停止服务 | `systemctl stop fan-files` |
| 实时日志 | `journalctl -u fan-files -f` |
| 最近 50 行 | `journalctl -u fan-files -n 50` |
| 开机自启 | `systemctl enable fan-files` |
| 禁用自启 | `systemctl disable fan-files` |

## 卸载

```bash
# 保留数据目录（推荐，重装即可恢复）
bash scripts/uninstall.sh -y --keep-data

# 连数据库一起删除
bash scripts/uninstall.sh -y --purge

# 远程卸载
curl -fsSL https://raw.githubusercontent.com/meimolihan/fan-files/main/scripts/uninstall.sh \
  | bash -s -- -y --keep-data
```

## 配置文件

生成位置：`/etc/fan-files/settings.json`（安装时自动生成，亦可手动编辑后 `systemctl restart fan-files`）

```json
{
  "port": 8678,
  "baseURL": "",
  "address": "0.0.0.0",
  "log": "stdout",
  "database": "/var/lib/fan-files/fan-files.db",
  "root": "/vol1/1000",
  "extraRoots": [
    {"path": "/vol2/1000", "label": "存储空间2"}
  ]
}
```

关键字段：
- `port`：监听端口（默认 8678）
- `root`：主存储根目录
- `extraRoots`：附加存储根数组，每项含 `path` 绝对路径与 `label` 显示名
- `database`：SQLite 数据库路径（含用户、设置、分享链接等）

## 发布流程

```bash
# 交互式：输入版本号，确认后构建、提交、打 tag、推送、创建 Release
./scripts/build-and-push.sh v1.0.1

# 全自动（需先 gh auth login）
./scripts/build-and-push.sh v1.0.1 --yes
```

产物：`bin/fan-files`、`bin/fan-files_linux_amd64`（上传至 GitHub Release）。

## 安全建议
- **不要直接暴露公网**：置于反向代理（Nginx/Caddy/Traefik）后，由代理终结 TLS 并做认证
- **禁用命令执行器**：默认关闭，不要开启 `--disable-exec=false`
- **以非特权用户、容器运行**：仅挂载需服务的目录
- **JWT 会话不可撤销**：密码修改/登出不会使已签发 token 失效，泄露视为有效至过期

## 许可证

[Apache License 2.0](LICENSE) © fan-files Contributors  
（原项目：File Browser Contributors）