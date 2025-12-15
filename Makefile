.PHONY: build run test clean deps

# 构建
build:
	@mkdir -p bin
	go build -o bin/nofx-go cmd/main.go

# 运行
run:
	go run cmd/main.go

# 测试
test:
	go test ./...

# 测试覆盖率
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# 清理
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# 安装依赖
deps:
	go mod download
	go mod tidy

# 格式化代码
fmt:
	go fmt ./...

# 代码检查
lint:
	golangci-lint run

# 运行所有检查
check: fmt lint test

# 加密工具
encrypt:
	go run ./cmd/encrypt

