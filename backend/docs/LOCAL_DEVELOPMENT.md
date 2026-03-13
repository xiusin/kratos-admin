# 本地开发指南 (Local Development Guide)

本文档介绍如何在**纯本地环境**（不使用 Docker）进行开发。

---

## 📋 前置条件

### 开发工具

1. **Go** (>= 1.21)
2. **buf** - Protobuf 构建工具
3. **protoc-gen-go** - Go Protobuf 插件
4. **protoc-gen-go-grpc** - Go gRPC 插件

**快速安装所有工具：**

```bash
cd backend
make install-tools
```

或手动安装：

```bash
# 安装 buf
go install github.com/bufbuild/buf/cmd/buf@latest

# 安装 Protobuf 插件
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# 确保 GOPATH/bin 在 PATH 中
export PATH="$PATH:$(go env GOPATH)/bin"
```

### 必需服务

1. **PostgreSQL** (端口 5432)
2. **Redis** (端口 6379)

### 可选服务

3. **MinIO** (端口 9000/9001) - 对象存储服务

---

## 🚀 快速开始

### 步骤 0: 安装开发工具（首次运行）

```bash
cd backend

# 方式 1: 自动安装所有工具（推荐）
make install-tools

# 方式 2: 检查工具是否已安装
make check-tools

# 方式 3: 只安装缺失的工具
make auto-install-tools
```

**注意：** 确保 `$GOPATH/bin` 在你的 `PATH` 中：

```bash
# 临时添加（当前终端有效）
export PATH="$PATH:$(go env GOPATH)/bin"

# 永久添加（添加到 ~/.bashrc 或 ~/.zshrc）
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.bashrc
source ~/.bashrc
```

### 步骤 1: 启动服务

### 方式 1: 使用 Makefile（推荐）

```bash
# 1. 检查本地服务状态
cd backend
make check-local

# 2. 启动服务
make run-local
```

### 方式 2: 使用启动脚本

```bash
# 启动服务（会自动检查依赖）
cd backend
bash scripts/local/start_local.sh
```

### 方式 3: 手动启动

```bash
cd backend/app/admin/service

# 生成代码
make api
make openapi

# 启动服务
make run
```

---

## ⚙️ 本地服务安装

### macOS (使用 Homebrew)

```bash
# 安装 PostgreSQL
brew install postgresql@15
brew services start postgresql@15

# 创建数据库
createdb gwa

# 安装 Redis
brew install redis
brew services start redis

# 配置 Redis 密码
redis-cli
> CONFIG SET requirepass "*Abcd123456"
> AUTH *Abcd123456
> CONFIG REWRITE
> exit

# 安装 MinIO (可选)
brew install minio/stable/minio
minio server /usr/local/var/minio --console-address ":9001"
```

### Linux (Ubuntu/Debian)

```bash
# 安装 PostgreSQL
sudo apt update
sudo apt install postgresql postgresql-contrib
sudo systemctl start postgresql

# 创建数据库
sudo -u postgres createdb gwa
sudo -u postgres psql -c "ALTER USER postgres PASSWORD '*Abcd123456';"

# 安装 Redis
sudo apt install redis-server
sudo systemctl start redis

# 配置 Redis 密码
sudo nano /etc/redis/redis.conf
# 找到 # requirepass foobared
# 改为 requirepass *Abcd123456
sudo systemctl restart redis

# 安装 MinIO (可选)
wget https://dl.min.io/server/minio/release/linux-amd64/minio
chmod +x minio
sudo mv minio /usr/local/bin/
minio server /data --console-address ":9001"
```

### 使用 Docker 运行依赖服务（推荐）

如果不想在本地安装这些服务，可以只用 Docker 运行依赖：

```bash
cd backend
make docker-libs
```

这会启动 PostgreSQL、Redis、MinIO 等依赖服务，但不启动应用本身。

---

## 🔧 配置文件

### 本地配置文件

本地开发使用 `data.local.yaml` 配置文件：

```yaml
# backend/app/admin/service/configs/data.local.yaml
data:
  database:
    driver: "postgres"
    source: "host=localhost port=5432 user=postgres password=*Abcd123456 dbname=gwa sslmode=disable"
    migrate: true
    debug: true

  redis:
    addr: "localhost:6379"
    password: "*Abcd123456"
```

### 自动生成本地配置

```bash
cd backend
make setup-local-config
```

这会自动创建 `data.local.yaml` 并将服务地址改为 `localhost`。

---

## 🐛 调试

### VSCode 调试配置

创建 `.vscode/launch.json`：

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Debug Admin Service",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/backend/app/admin/service/cmd/server",
      "args": ["-c", "${workspaceFolder}/backend/app/admin/service/configs"],
      "cwd": "${workspaceFolder}/backend/app/admin/service",
      "env": {},
      "showLog": true
    }
  ]
}
```

然后按 `F5` 启动调试。

### GoLand/IntelliJ IDEA 调试配置

1. 打开 `Run` → `Edit Configurations`
2. 点击 `+` → `Go Build`
3. 配置：
   - **Name**: Debug Admin Service
   - **Run kind**: Directory
   - **Directory**: `backend/app/admin/service/cmd/server`
   - **Working directory**: `backend/app/admin/service`
   - **Program arguments**: `-c ./configs`
4. 点击 `Debug` 按钮启动调试

---

## 📊 验证服务

### 检查服务状态

```bash
# 检查 HTTP 服务
curl http://localhost:7788/health

# 检查 Swagger 文档
open http://localhost:7788/docs/

# 检查 OpenAPI 文档
curl http://localhost:7788/docs/openapi.yaml
```

### 检查依赖服务

```bash
cd backend
make check-local
```

输出示例：
```
🔍 Checking local services...

PostgreSQL:
✅ PostgreSQL is running

Redis:
✅ Redis is running

MinIO:
✅ MinIO is running
```

---

## 🔄 开发工作流

### 典型开发流程

```bash
# 1. 启动依赖服务（只需一次）
cd backend
make docker-libs

# 2. 修改代码
vim app/admin/service/internal/service/user.go

# 3. 重新生成代码（如果修改了 Protobuf）
cd app/admin/service
make api
make openapi

# 4. 启动服务
make run

# 5. 测试 API
curl -X POST http://localhost:7788/admin/v1/users \
  -H "Content-Type: application/json" \
  -d '{"username":"test","email":"test@example.com"}'
```

### 热重载开发

使用 `air` 实现热重载：

```bash
# 安装 air
go install github.com/cosmtrek/air@latest

# 在服务目录下运行
cd backend/app/admin/service
air
```

创建 `.air.toml` 配置文件：

```toml
root = "."
tmp_dir = "tmp"

[build]
  cmd = "go build -o ./tmp/main ./cmd/server"
  bin = "tmp/main"
  args_bin = ["-c", "./configs"]
  include_ext = ["go", "yaml"]
  exclude_dir = ["tmp", "vendor"]
  delay = 1000
```

---

## 🛠️ 常见问题

### Q0: buf: command not found

**错误信息：**
```
/bin/sh: buf: command not found
make[1]: *** [api] Error 127
```

**解决方法：**

```bash
# 方式 1: 自动安装所有工具
cd backend
make install-tools

# 方式 2: 只安装 buf
go install github.com/bufbuild/buf/cmd/buf@latest

# 方式 3: 使用 Homebrew (macOS)
brew install bufbuild/buf/buf

# 确保 GOPATH/bin 在 PATH 中
export PATH="$PATH:$(go env GOPATH)/bin"

# 验证安装
buf --version
```

**永久解决：** 将 GOPATH/bin 添加到 shell 配置文件

```bash
# Bash
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.bashrc
source ~/.bashrc

# Zsh
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.zshrc
source ~/.zshrc
```

### Q1: PostgreSQL 连接失败

**错误信息：**
```
failed to connect to database: dial tcp [::1]:5432: connect: connection refused
```

**解决方法：**
```bash
# 检查 PostgreSQL 是否运行
pg_isready -h localhost -p 5432

# 如果未运行，启动服务
# macOS
brew services start postgresql@15

# Linux
sudo systemctl start postgresql
```

### Q2: Redis 认证失败

**错误信息：**
```
NOAUTH Authentication required
```

**解决方法：**
```bash
# 配置 Redis 密码
redis-cli
> CONFIG SET requirepass "*Abcd123456"
> AUTH *Abcd123456
> CONFIG REWRITE
> exit

# 或修改配置文件
sudo nano /etc/redis/redis.conf
# 添加: requirepass *Abcd123456
sudo systemctl restart redis
```

### Q3: 端口被占用

**错误信息：**
```
bind: address already in use
```

**解决方法：**
```bash
# 查找占用端口的进程
lsof -i :7788

# 杀死进程
kill -9 <PID>
```

### Q4: 数据库迁移失败

**解决方法：**
```bash
# 删除数据库重新创建
dropdb gwa
createdb gwa

# 或手动运行迁移
cd backend/app/admin/service
go run ./cmd/server -c ./configs
```

---

## 📚 相关文档

- [后端 README](../README.md)
- [API 文档](http://localhost:7788/docs/)
- [Docker 部署指南](./DOCKER_DEPLOYMENT.md)

---

## 💡 提示

1. **推荐使用混合模式**：Docker 运行依赖，本地运行应用
   ```bash
   make docker-libs  # 启动依赖
   make run-local    # 启动应用
   ```

2. **使用 IDE 调试**：在 VSCode 或 GoLand 中设置断点调试

3. **热重载开发**：使用 `air` 实现代码修改自动重启

4. **配置管理**：使用 `data.local.yaml` 管理本地配置，不要提交到 Git

---

## 🎯 下一步

- [实现用户管理功能](../app/admin/service/README.md)
- [编写单元测试](./TESTING.md)
- [API 开发指南](./API_DEVELOPMENT.md)
