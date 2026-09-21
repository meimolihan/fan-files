## Multistage build: frontend -> static Go binary -> busybox runtime
## 使 docker buildx 可以直接交叉构建 linux/amd64 与 linux/arm64 镜像
## （无需预先把宿主二进制放进构建上下文）。

# ============================================================
# Stage 1: 前端构建（Vue 3 + pnpm，产物 frontend/dist 由 Go embed）
# ============================================================
FROM node:24-alpine AS frontend

WORKDIR /app/frontend

# 只复制依赖清单，命中 Docker 缓存时跳过 pnpm install
COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN corepack enable && pnpm install --frozen-lockfile --prefer-offline

# 复制前端源码并构建（含 typecheck/vite build）
COPY frontend/ ./
RUN pnpm run build

# ============================================================
# Stage 2: Go 静态编译（CGO_ENABLED=0，含前端 dist）
# ============================================================
FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache ca-certificates

# 构建时注入 commit（版本号来自 version/version.go，无需注入）
ARG GIT_COMMIT="unknown"

# 先只复制 go.mod/go.sum，命中缓存时复用依赖层
COPY go.mod go.sum ./
RUN go mod download

# 前端产物必须在 go build 前就位（frontend/assets.go embed dist/*）
COPY --from=frontend /app/frontend/dist ./frontend/dist

COPY . .

RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags="-s -w -X github.com/meimolihan/fan-files/version.CommitSHA=${GIT_COMMIT}" \
    -o /out/fan-files .

# ============================================================
# Stage 3: 中间引用文件（ca-certificates/mailcap/tini/JSON.sh）
# ============================================================
FROM alpine:3.23 AS fetcher

# install and copy ca-certificates, mailcap, and tini-static; download JSON.sh
RUN apk update && \
    apk --no-cache add ca-certificates mailcap tini-static && \
    wget -O /JSON.sh https://raw.githubusercontent.com/dominictarr/JSON.sh/0d5e5c77365f63809bf6e77ef44a1f34b0e05840/JSON.sh

## Stage 4: Use lightweight BusyBox image for final runtime environment
FROM busybox:1.37.0-musl

# Define non-root user UID and GID
ENV UID=1000
ENV GID=1000

# Create user group and user
RUN addgroup -g $GID user && \
    adduser -D -u $UID -G user user

# Copy binary, scripts, and configurations into image with proper ownership
COPY --chown=user:user --from=builder /out/fan-files /bin/fan-files
COPY --chown=user:user docker/common/ /
COPY --chown=user:user docker/alpine/ /
COPY --chown=user:user --from=fetcher /sbin/tini-static /bin/tini
COPY --from=fetcher /JSON.sh /JSON.sh
COPY --from=fetcher /etc/ca-certificates.conf /etc/ca-certificates.conf
COPY --from=fetcher /etc/ca-certificates /etc/ca-certificates
COPY --from=fetcher /etc/mime.types /etc/mime.types
COPY --from=fetcher /etc/ssl /etc/ssl

# Create data directories, set ownership, and ensure healthcheck script is executable
RUN mkdir -p /config /database /srv && \
    chown -R user:user /config /database /srv \
    && chmod +x /healthcheck.sh /init.sh

# Define healthcheck script
HEALTHCHECK --start-period=2s --interval=5s --timeout=3s CMD /healthcheck.sh

# Set the user, volumes and exposed ports
USER user

VOLUME /srv /config /database

EXPOSE 80

ENTRYPOINT [ "tini", "--", "/init.sh" ]