# 贡献指南

感谢您对NOFX-Go项目的关注！我们欢迎所有形式的贡献。

## 如何贡献

### 报告问题

1. 检查[Issues](https://github.com/yuechangmingzou/nofx-go/issues)中是否已有相关问题
2. 创建新Issue，详细描述问题
   - 问题描述
   - 复现步骤
   - 预期行为
   - 实际行为
   - 环境信息（Go版本、操作系统等）

### 提交代码

1. **Fork项目**
   ```bash
   git clone https://github.com/yuechangmingzou/nofx-go.git
   cd nofx-go
   ```

2. **创建分支**
   ```bash
   git checkout -b feature/your-feature-name
   # 或
   git checkout -b fix/your-bug-fix
   ```

3. **进行修改**
   - 遵循Go代码规范
   - 添加必要的注释
   - 更新相关文档
   - 添加必要的测试

4. **提交代码**
   ```bash
   git add .
   git commit -m "feat: 添加新功能"  # 或 fix:, docs:, style:, refactor:, test:, chore:
   git push origin feature/your-feature-name
   ```

5. **提交Pull Request**
   - 在GitHub上创建Pull Request
   - 详细描述您的更改
   - 关联相关Issue（如果有）

## 代码规范

### 提交信息格式

遵循[Conventional Commits](https://www.conventionalcommits.org/zh-hans/v1.0.0/)规范：

- `feat:` 新功能
- `fix:` Bug修复
- `docs:` 文档更新
- `style:` 代码格式调整（不影响功能）
- `refactor:` 代码重构
- `test:` 测试相关
- `chore:` 构建/工具相关

示例：
```
feat: 添加WebSocket Origin验证功能
fix: 修复CancelOrder接口签名不匹配
docs: 更新README，添加配置说明
```

### 代码风格

- 使用 `go fmt` 格式化代码
- 遵循Go代码规范
- 使用有意义的变量和函数名
- 添加必要的注释
- 保持函数简洁（建议不超过100行）

### 测试

- 为新功能添加测试
- 确保所有测试通过
- 运行 `go test ./...` 验证

## 开发环境

1. 安装Go 1.23.0或更高版本
2. 安装Redis
3. 克隆项目
4. 复制 `.env.example` 为 `.env` 并配置
5. 运行 `go mod download` 安装依赖

## 问题反馈

如果您有任何问题或建议，欢迎：

- 提交Issue
- 创建Pull Request

感谢您的贡献！
