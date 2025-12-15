# NOFX Go - 量化交易机器人

[![Go Version](https://img.shields.io/badge/Go-1.23.0+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Go语言实现的量化交易机器人，支持AI交易和规则策略两种模式。具备完整的交易闭环、分布式锁、去重机制、止损止盈守护等企业级特性。

## 🎯 项目特性

- ✅ **完整的交易闭环** - 从市场扫描到信号生成、订单执行、止损止盈守护全流程覆盖
- ✅ **AI/规则双模式** - 支持DeepSeek、OpenAI、Gemini等多种AI提供商，也支持基于技术指标的规则策略
- ✅ **企业级特性** - 分布式锁、信号去重、审计日志、性能监控
- ✅ **安全可靠** - API密钥加密存储、WebSocket认证、CORS配置
- ✅ **代码质量** - 模块化设计、接口抽象、工具函数复用、遵循DRY原则
- ✅ **实时监控** - Web Dashboard、WebSocket实时推送、性能指标收集

## 📋 主要功能

### 核心功能

#### 市场扫描与信号生成
- **市场扫描器**: 自动扫描市场，识别波动最大的币种，支持流式扫描和并发处理
- **AI交易员**: 支持DeepSeek、OpenAI、Gemini等多种AI提供商，智能决策交易
- **规则策略**: 基于EMA、RSI、布林带等技术指标的规则交易策略
- **信号去重**: 基于时间窗口的信号去重机制，防止重复下单

#### 订单执行
- **执行引擎**: 分布式锁、去重检查、审计日志、异步订单确认
- **止损止盈守护**: 自动管理止损止盈订单，支持分批止盈（TP1/TP2）
- **保护信息管理**: Redis存储保护信息，自动清理已平仓的保护数据

#### Web服务
- **HTTP API**: 完整的RESTful API，支持状态查询、持仓管理、配置管理
- **WebSocket**: 实时市场数据推送，支持状态、持仓、余额、市场数据四种消息类型
- **性能监控**: 自动收集HTTP请求、WebSocket连接、系统资源、业务指标
- **配置优化**: 根据性能自动优化配置参数

## 🚀 快速开始

### 前置要求

- **Go 1.23.0+** - 推荐使用最新版本
- **Redis 6.0+** - 用于队列、缓存、分布式锁
- **Binance API密钥** - 可选，支持`DRY_RUN=true`模拟模式

### 安装

```bash
# 克隆仓库
git clone https://github.com/yuechangmingzou/nofx-go.git
cd nofx-go

# 安装依赖
go mod download

# 配置环境变量
cp .env.example .env
# 编辑 .env 文件，填入您的配置
```

### 快速运行

```bash
# 方式1: 直接运行
go run ./cmd/main.go

# 方式2: 编译后运行
go build -o bin/nofx-go ./cmd/main.go
./bin/nofx-go

# 方式3: 使用Makefile
make build
make run

# 方式4: Docker 一键部署（推荐用于生产环境）
# 适用于 Linux Ubuntu 22.04 64位操作系统
chmod +x deploy.sh
bash deploy.sh
# 详细说明请参考 docs/DOCKER_DEPLOY.md
```

## 📁 项目结构

```
nofx-go/
├── cmd/                    # 主程序入口
│   ├── main.go            # 主程序
│   ├── encrypt/           # API密钥加密工具
│   └── loadtest/          # 压力测试工具
├── internal/              # 内部包
│   ├── ai/                # AI交易员（DeepSeek/OpenAI/Gemini）
│   ├── bot/               # 交易机器人核心逻辑
│   ├── config/            # 配置管理（加载、验证、优化）
│   ├── exchange/          # 交易所接口（Binance实现）
│   ├── execution/         # 执行引擎（订单执行、守护进程）
│   ├── indicators/        # 技术指标计算（EMA/RSI/BB等）
│   ├── metrics/           # 性能监控指标收集
│   ├── scanner/           # 市场扫描器（流式扫描、符号池）
│   ├── strategies/        # 交易策略（规则策略）
│   ├── utils/             # 工具函数（加密、日志、Redis、工具类）
│   └── web/               # Web服务（HTTP API、WebSocket、Dashboard）
├── pkg/                   # 公共包
│   └── types/             # 类型定义（Exchange接口、数据结构）
├── docs/                  # 文档
│   └── API_KEY_ENCRYPTION.md  # API密钥加密指南
├── tests/                 # 测试
├── strategies/            # 策略文件
├── web/                   # Web静态资源
│   ├── static/            # 静态文件
│   └── templates/         # 模板文件（Dashboard）
├── .github/               # GitHub配置
│   └── workflows/         # CI/CD
├── Makefile               # 构建脚本
├── Dockerfile             # Docker镜像
├── docker-compose.yml     # Docker Compose配置
├── go.mod                 # Go模块定义
├── README.md              # 项目说明
├── CHANGELOG.md           # 变更日志
├── CONTRIBUTING.md        # 贡献指南
└── LICENSE                # 许可证
```

## ⚙️ 配置

### 主要配置项

通过环境变量或`.env`文件配置：

#### 基础配置
- `REDIS_HOST`: Redis主机（默认: localhost）
- `REDIS_PORT`: Redis端口（默认: 6379）
- `DRY_RUN`: 是否启用模拟模式（默认: true）
- `LOG_LEVEL`: 日志级别（DEBUG/INFO/WARN/ERROR）

#### Binance配置
- `BINANCE_API_KEY`: Binance API密钥（支持加密存储）
- `BINANCE_SECRET_KEY`: Binance密钥（支持加密存储）
- `BINANCE_TESTNET`: 是否使用测试网（默认: false）

#### AI配置
- `AI_PROVIDER`: AI提供商（deepseek/openai/gemini）
- `DEEPSEEK_API_KEY`: DeepSeek API密钥（支持加密存储）
- `OPENAI_API_KEY`: OpenAI API密钥（支持加密存储）
- `GEMINI_API_KEY`: Gemini API密钥（支持加密存储）
- `AI_TEMPERATURE`: AI温度参数（默认: 0.3）
- `AI_MAX_TOKENS`: AI最大token数（默认: 4000）

#### 交易配置
- `MAX_NOTIONAL_PER_TRADE`: 单笔交易最大名义价值（默认: 50 USDT）
- `MAX_CONCURRENT_POSITIONS`: 最大并发持仓数（默认: 5）
- `STRAT_DEFAULT_NOTIONAL_USDT`: 默认交易金额（默认: 50 USDT）

完整配置项请参考 `.env.example` 或 `internal/config/config.go`

### API 密钥加密存储

为了增强安全性，支持对 API 密钥进行 AES-256-GCM 加密存储：

```bash
# 1. 生成加密密钥（32字节或更长）
export ENCRYPTION_KEY=$(openssl rand -base64 32)

# 2. 加密 API 密钥（交互式，推荐）
go run ./cmd/encrypt -key=BINANCE_API_KEY -i

# 或使用Makefile
make encrypt

# 3. 在 .env 中使用加密值
BINANCE_API_KEY=encrypted:xxxxx
BINANCE_SECRET_KEY=encrypted:yyyyy
```

**⚠️ 重要提示**：
- 生产环境必须设置 `ENCRYPTION_KEY` 环境变量
- 不要将 `ENCRYPTION_KEY` 提交到版本控制系统
- 加密后的值以 `encrypted:` 开头

详细说明请参考 [API 密钥加密存储指南](docs/API_KEY_ENCRYPTION.md)

## 📊 性能监控

系统自动收集以下指标：
- **HTTP请求统计** - 请求数、成功率、响应时间
- **WebSocket连接统计** - 连接数、消息数、成功率
- **系统资源使用** - CPU、内存、Goroutine数量
- **业务指标** - 信号数、订单数、AI请求数、成功率

查看指标：
```bash
# 查看性能指标
redis-cli GET "nofx:metrics:performance"

# 查看AI统计
redis-cli GET "nofx:ai_api_stats"
```

## 🧪 测试

### 单元测试
```bash
# 运行所有测试
go test ./...

# 运行测试并查看覆盖率
make test-coverage
```

### 压力测试
```bash
# 使用Go测试框架
go test ./tests -run TestLoadTest -v

# 或使用独立工具
go run ./cmd/loadtest/main.go -url http://localhost:8000 -c 10 -n 1000
```

## 🛠️ 开发工具

### Makefile命令
```bash
make build          # 编译
make run            # 运行
make test           # 测试
make lint           # 代码检查
make fmt            # 格式化代码
make clean          # 清理构建文件
make encrypt        # 加密工具
```

### 代码质量
- 使用 `golangci-lint` 进行代码检查
- 遵循 Go 代码规范
- 使用 `EditorConfig` 统一编辑器配置

## 📚 文档

- [API 密钥加密存储指南](docs/API_KEY_ENCRYPTION.md) - 详细的加密存储使用说明
- [变更日志](CHANGELOG.md) - 版本更新记录
- [贡献指南](CONTRIBUTING.md) - 如何参与项目贡献

## 🔒 安全特性

- ✅ API密钥加密存储（AES-256-GCM）
- ✅ WebSocket Token认证
- ✅ CORS配置支持
- ✅ 分布式锁防止并发下单
- ✅ 信号去重机制
- ✅ 审计日志记录

## 📝 许可证

本项目采用 [MIT License](LICENSE) 许可证。

## 🤝 贡献

欢迎提交Issue和Pull Request！

在提交PR之前，请：
1. 阅读 [贡献指南](CONTRIBUTING.md)
2. 确保代码通过 `make lint` 检查
3. 添加必要的测试
4. 更新相关文档

## 📧 联系方式

- GitHub: [yuechangmingzou/nofx-go](https://github.com/yuechangmingzou/nofx-go)
- Issues: [提交问题](https://github.com/yuechangmingzou/nofx-go/issues)

---

**⚠️ 风险提示**: 量化交易存在风险，请谨慎使用。建议先在测试环境或模拟模式下充分测试，再考虑实盘交易。
