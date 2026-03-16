# 1. 进入 cmd/server 目录
cd ~/Desktop/kratos-admin/backend/app/consumer/service/cmd/server

# 2. 重新生成 Wire 代码
go generate

# 3. 返回 service 根目录
cd ~/Desktop/kratos-admin/backend/app/consumer/service

# 4. 编译验证
go build ./...
