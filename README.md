# fan-files

轻量级文件管理器，基于 filebrowser 二次开发。支持**多根存储合并**、Web 界面上传/下载/预览/编辑，部署即用。

## 特性
- **多根合并**：主存储根 + 多个附加存储根，首页合并列表，`storage` 字段区分来源
- **原生 systemd**：一键安装/卸载/升级，开机自启，日志统一管理
- **零依赖静态二进制**：单文件部署，无运行时依赖
- **远程一键安装**：`curl | bash` 无需本地构建，自动从 Release 下载对应架构二进制
- **保留数据升级**：卸载保留 `/var/lib/fan-files`，重装即可恢复用户/设置/密码
- **备份与还原**：内置备份/还原功能，支持 Web UI、CLI 脚本、API 调用，自动清理旧备份

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
cd /vol1/1000/compose/opencode/workspace/fan-files
cd frontend && pnpm install --frozen-lockfile && pnpm run build && cd ..
go build -ldflags='-s -w' -o fan-files .

mkdir -p /var/lib/fan-files
bash scripts/install.sh -y -p 8678 -d /var/lib/fan-files -r /vol1/1000 --extra-root "存储空间2=/vol2/1000"
```

服务器监听:8678
根存储：/vol1/1000
额外存储：（/vol2/1000标签：“存储空间2”）
安装目录：/var/lib/fan-files

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

## CLI 管理命令

二进制自带管理命令（`uninstall` 需 root）：

| 命令 | 说明 |
|------|------|
| `fan-files status` | 显示运行方式（systemd / Docker / 直接运行）、PID、监听端口、运行时长、内存、数据目录 |
| `fan-files uninstall [-y] [--purge\|--keep-data]` | 停止并移除服务/容器/进程，删除二进制与安装记录；可选删除数据目录 |
| `fan-files --version` / `fan-files version` | 显示版本号 |

```bash
fan-files status
sudo fan-files uninstall -y            # 免确认卸载，保留数据目录
sudo fan-files uninstall -y --purge    # 免确认卸载，并删除数据目录
```

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

## 备份与还原

fan-files 内置完整的备份/还原机制，支持三种方式：

### 1. Web UI（推荐，管理员可见）
登录后进入 **设置 → 备份与还原**，可：
- 查看备份列表（名称、大小、时间）
- 一键创建备份（自动停止服务、打包、清理旧备份、重启服务）
- 选择备份文件还原（自动停止服务、解包、重启服务）
- 删除不需要的备份
- 实时显示任务进度

### 2. CLI 脚本（适合定时任务）
```bash
# 创建备份（保留最近 6 份，自定义目录）
bash scripts/fan-files_backup.sh 6 /vol2/1000/file/backup/fan-files-backup

# 还原最新备份（默认目录）
bash scripts/fan-files_recover.sh

# 还原指定目录最新备份
bash scripts/fan-files_recover.sh /path/to/backup
```

脚本特性：
- 自动读取 `/etc/fan-files.conf` 获取数据目录
- 停止服务 → 打包/解包 → 清理旧备份 → 启动服务
- 保留份数可配置，默认 6 份
- 彩色输出，进度可视

### 3. REST API（适合自动化集成）
```bash
# 获取备份列表
curl -H "X-Auth: $TOKEN" http://localhost:8678/api/backup

# 创建备份（异步）
curl -X POST -H "X-Auth: $TOKEN" -H "Content-Type: application/json" \
  -d '{"keepNum":6,"async":true}' http://localhost:8678/api/backup

# 还原指定备份
curl -X POST -H "X-Auth: $TOKEN" -H "Content-Type: application/json" \
  -d '{"file":"FanFiles-2026-09-12_18-30-00.tar.gz"}' http://localhost:8678/api/backup/restore

# 查询任务状态
curl -H "X-Auth: $TOKEN" http://localhost:8678/api/backup/job/backup-xxx

# 删除备份
curl -X DELETE -H "X-Auth: $TOKEN" http://localhost:8678/api/backup/FanFiles-xxx.tar.gz
```

### 备份文件格式
- 存放目录：`{数据目录}/backup/`（默认 `/var/lib/fan-files/backup/`，可配置）
- 文件名：`FanFiles-YYYY-MM-DD_HH-MM-SS.tar.gz`
- 包含：SQLite 数据库、配置文件、用户数据等

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

# 全自动（需先 gh auth login），-m 可附加发版备注
./scripts/build-and-push.sh v1.0.1 --yes -m "本次更新内容"
```

产物：`bin/fan-files`、`bin/fan-files_linux_amd64`（上传至 GitHub Release）。

## 安全建议
- **不要直接暴露公网**：置于反向代理（Nginx/Caddy/Traefik）后，由代理终结 TLS 并做认证
- **禁用命令执行器**：默认关闭，不要开启 `--disable-exec=false`
- **以非特权用户、容器运行**：仅挂载需服务的目录
- **JWT 会话不可撤销**：密码修改/登出不会使已签发 token 失效，泄露视为有效至过期

## 常用命令速查表

| 场景 | 命令 |
|------|------|
| **安装** | `bash scripts/install.sh -y -p 8678 -d /var/lib/fan-files -r /vol1/1000 --extra-root "存储空间2=/vol2/1000"` |
| **远程安装** | `curl -fsSL https://raw.githubusercontent.com/meimolihan/fan-files/main/scripts/install.sh \| bash -s -- -y -p 8678 -d /var/lib/fan-files -r /vol1/1000 --extra-root "存储空间2=/vol2/1000"` |
| **交互式安装** | `bash scripts/install.sh` |
| **查看服务状态** | `systemctl status fan-files` |
| **重启服务** | `systemctl restart fan-files` |
| **停止服务** | `systemctl stop fan-files` |
| **实时日志** | `journalctl -u fan-files -f` |
| **查看最近 50 行日志** | `journalctl -u fan-files -n 50` |
| **重置用户密码（脚本，自动 sudo）** | `sudo scripts/reset-password.sh <用户名> '<新密码>'` |
| **重置用户密码（CLI 单条命令）** | `sudo /usr/local/bin/fan-files -c /etc/fan-files/settings.json users update <用户名> -p '<新密码>'` |
| **卸载（保留数据）** | `bash scripts/uninstall.sh -y --keep-data` |
| **卸载（删除数据）** | `bash scripts/uninstall.sh -y --purge` |
| **远程卸载** | `curl -fsSL https://raw.githubusercontent.com/meimolihan/fan-files/main/scripts/uninstall.sh \| bash -s -- -y --keep-data` |
| **创建备份（CLI，保留 6 份）** | `bash scripts/fan-files_backup.sh 6 /vol2/1000/file/backup/fan-files-backup` |
| **还原最新备份（CLI）** | `bash scripts/fan-files_recover.sh` |
| **还原指定目录备份（CLI）** | `bash scripts/fan-files_recover.sh /path/to/backup` |
| **查看备份列表（API）** | `curl -H "X-Auth: $TOKEN" http://localhost:8678/api/backup` |
| **创建备份（API，异步）** | `curl -X POST -H "X-Auth: $TOKEN" -H "Content-Type: application/json" -d '{"keepNum":6,"async":true}' http://localhost:8678/api/backup` |
| **还原指定备份（API）** | `curl -X POST -H "X-Auth: $TOKEN" -H "Content-Type: application/json" -d '{"file":"FanFiles-xxx.tar.gz"}' http://localhost:8678/api/backup/restore` |
| **查询备份任务状态（API）** | `curl -H "X-Auth: $TOKEN" http://localhost:8678/api/backup/job/backup-xxx` |
| **删除备份（API）** | `curl -X DELETE -H "X-Auth: $TOKEN" http://localhost:8678/api/backup/FanFiles-xxx.tar.gz` |
| **构建并发布** | `./scripts/build-and-push.sh v1.0.1 --yes` |
| **前端构建** | `cd frontend && pnpm install && CI=true pnpm run build` |
| **后端构建** | `export PATH=$PATH:/usr/local/go/bin && CGO_ENABLED=0 go build -ldflags="-s -w -X 'github.com/meimolihan/fan-files/version.Version=1.0.0' -X 'github.com/meimolihan/fan-files/version.CommitSHA=$(git rev-parse HEAD)'" -o fan-files .` |

## 许可证

[Apache License 2.0](LICENSE) © fan-files Contributors  
（原项目：File Browser Contributors）