# 变更日志

本项目遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/) 规范，版本号遵循 [SemVer](https://semver.org/lang/zh-CN/) 规范。

## [未发布]

### 已修复
- 修复CancelOrder接口签名不匹配问题
- 修复GetOrder接口签名不匹配问题
- 修复Order结构体缺少ReduceOnly字段问题
- 修复Redis连接panic问题，改为logger.Error
- 修复guard.go中使用redis.Keys的问题，改为redis.Scan
- 修复WebSocket Origin验证
- 修复guard.go中调用CancelOrder的问题
- 修复GetPositions中symbol类型断言不健壮的问题
- 修复文件编码问题，移除BOM标记

### 已添加
- 添加策略文件 `strategies/顺势狙击手.txt`
- 添加Dashboard界面 `web/templates/dashboard.html`
- 添加Docker配置文件 Dockerfile, docker-compose.yml, .dockerignore
- 添加LICENSE文件 MIT License
- 添加配置文件示例 `.env.example`
- 添加代码质量工具配置 `.golangci.yml`, `.editorconfig`
- 添加CI/CD配置 `.github/workflows/ci.yml`
- 添加API密钥加密存储功能
- 添加公共工具包（maputil, contextutil, jsonutil, parseutil, positionutil）
- 添加.gitattributes文件，确保文本文件正确显示

### 已改进
- 改进Makefile，添加bin目录到.gitignore
- 改进代码结构，消除重复代码，遵循DRY原则
- 改进错误处理，统一使用utils包的工具函数
- 改进代码质量，统一context创建模式

---

## [1.0.0] - 2025-01-12

### 初始发布
- 完成Go语言转换
- 实现AI交易功能
- 实现Binance API完整对接
- 实现Web服务和WebSocket功能
- 实现性能监控和指标收集
