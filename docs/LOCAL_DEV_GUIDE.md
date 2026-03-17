# GoWind Admin 本地开发指南

> macOS 纯本地环境，无需 Docker

---

## 一、环境准备（首次）

### 1. 基础工具

| 工具 | 版本要求 | 安装 |
|------|---------|------|
| Go | >= 1.21 | https://go.dev/dl/ |
| Node.js | >= 20.10.0 | https://nodejs.org/ |
| pnpm | >= 9.12.0 | `npm install -g pnpm` |

### 2. 安装基础设施（macOS Homebrew）

```bash
# PostgreSQL
brew install postgresql@15
brew services start postgresql@15

# 创建数据库
createdb gwa

# 设置 postgres 用户密码
psql postgres -c "ALTER USER postgres PASSWORD '*Abcd123456';"

# Redis
brew install redis
brew services start redis

# 设置 Redis 密码
redis-cli CONFIG SET requirepass "*Abcd123456"
redis-cli -a "*Abcd123456" CONFIG REWRITE

# MinIO（可选，文件上传功能需要）
brew install minio/stable/minio
mkdir -p ~/minio-data
# 启动 MinIO（需要手动在终端运行）
# MINIO_ROOT_USER=root MINIO_ROOT_PASSWORD='*Abcd123456' minio server ~/minio-data --console-address ":9001"
```

### 3. 安装 Go 开发工具链

```bash
cd backend

# 一键安装所有工具（buf、protoc 插件、ent、wire 等）
make init
```

确保 `$GOPATH/bin` 在 PATH 中：

```bash
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.zshrc
source ~/.zshrc
```

---

## 二、验证本地服务

```bash
cd backend
make check-local
```

输出应该是：
```
✅ PostgreSQL is running
✅ Redis is running
```

---

## 三、后端代码生成

### 代码生成顺序

```
.proto 文件  →  buf generate  →  api/gen/go/
Ent Schema   →  ent generate  →  internal/data/ent/
Wire         →  go generate   →  cmd/server/wire_gen.go
```

### 3.1 生成 Protobuf Go 代码

修改了 `backend/api/protos/` 下的 `.proto` 文件后：

```bash
cd backend && make api
```

### 3.2 生成 OpenAPI 文档

```bash
cd backend && make openapi
```

### 3.3 生成 Ent ORM 代码

修改了 `internal/data/ent/schema/` 下的 Schema 后：

```bash
cd backend/app/admin/service && make ent
```

### 3.4 生成 Wire 依赖注入代码

新增 Service/Repository 或修改了构造函数签名后：

```bash
cd backend/app/admin/service && make wire
```

### 3.5 一键生成所有代码

```bash
cd backend && make gen
# 等价于: ent + wire + api + openapi
```

---

## 四、启动后端服务

```bash
cd backend/app/admin/service
make run
```

或者从 backend 根目录：

```bash
cd backend && make run-local
```

### 后端服务地址

| 服务 | 地址 |
|------|------|
| Admin HTTP API | http://localhost:7788 |
| Swagger UI | http://localhost:7788/docs/ |
| OpenAPI YAML | http://localhost:7788/docs/openapi.yaml |

默认账号：`admin` / `admin`

---

## 五、启动前端

```bash
cd frontend

# 首次安装依赖
pnpm install

# 启动 Admin 前端
pnpm dev:antd
```

前端地址：http://localhost:5666

---

## 六、日常开发（快速参考）

```bash
# 终端 1 - 后端
cd backend/app/admin/service && make run

# 终端 2 - 前端
cd frontend && pnpm dev:antd
```

---

## 七、新功能开发流程

```bash
# 1. 改 proto 定义
vim backend/api/protos/admin/service/v1/xxx.proto
cd backend && make api

# 2. 改 Ent Schema（新表）
vim backend/app/admin/service/internal/data/ent/schema/xxx.go
cd backend/app/admin/service && make ent

# 3. 实现 Repository
vim backend/app/admin/service/internal/data/xxx_repo.go

# 4. 实现 Service
vim backend/app/admin/service/internal/service/xxx_service.go

# 5. 注册到 ProviderSet 和 RestServer 后重新生成 Wire
cd backend/app/admin/service && make wire

# 6. 编译验证
go build ./...
```

---

## 八、常用命令速查

```bash
# 生成所有代码
cd backend && make gen

# 编译验证
cd backend/app/admin/service && go build ./...

# 清理编译缓存（遇到奇怪错误时）
cd backend && go clean -cache

# 检查本地服务状态
cd backend && make check-local
```

---

## 九、本地配置文件

Admin Service 本地开发读取 `configs/data.local.yaml`（连接 localhost）：

```yaml
# backend/app/admin/service/configs/data.local.yaml
data:
  database:
    driver: "postgres"
    source: "host=localhost port=5432 user=postgres password=*Abcd123456 dbname=gwa sslmode=disable"
  redis:
    addr: "localhost:6379"
    password: "*Abcd123456"
```

---

## 十、故障排查

**`undefined: initApp`**
```bash
cd backend/app/admin/service/cmd/server && go generate
cd ../.. && go build ./...
```

**PostgreSQL 连接失败**
```bash
brew services restart postgresql@15
pg_isready -h localhost -p 5432
```

**Redis 认证失败**
```bash
redis-cli CONFIG SET requirepass "*Abcd123456"
redis-cli -a "*Abcd123456" CONFIG REWRITE
```

**go.sum 验证失败**
```bash
cd backend && go clean -modcache && go mod tidy
```

**端口被占用**
```bash
lsof -i :7788
kill -9 <PID>
```
