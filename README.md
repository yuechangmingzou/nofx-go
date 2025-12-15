# NOFX Go - 量化交易机器人

Go语言实现的量化交易机器人，支持AI交易和规则策略两种模式。

## 🎯 项目状态

**完成度: 100%** ✅

- ✅ 基础设施（配置、日志、Redis、类型定义）
- ✅ 交易所接口（Binance API完整实现）
- ✅ 核心交易功能（执行引擎、止损止盈守护、信号处理）
- ✅ Web服务（HTTP API、WebSocket、性能监控）
- ✅ 性能优化（指标收集、压力测试、配置优化）

## 📋 主要功能

### 核心功能
- **市场扫描器**: 自动扫描市场，识别波动最大的币种
- **AI交易员**: 支持DeepSeek、OpenAI、Gemini等多种AI提供商
- **规则策略**: 基于技术指标的规则交易策略
- **执行引擎**: 分布式锁、去重检查、审计日志
- **止损止盈守护**: 自动管理止损止盈订单

### Web服务
- **HTTP API**: 完整的RESTful API
- **WebSocket**: 实时市场数据推送
- **性能监控**: 自动收集性能指标
- **配置优化**: 根据性能自动优化配置

## 🚀 快速开始

### 前置要求
- Go 1.21+
- Redis
- Binance API密钥（可选，支持DRY_RUN模式）

### 安装

```bash
# 克隆仓库
git clone https://github.com/YOUR_USERNAME/nofx-go.git
cd nofx-go

# 安装依赖
go mod download

# 配置环境变量
cp .env.example .env
# 编辑 .env 文件，填入您的配置
```

### 运行

```bash
# 编译
go build -o nofx-go ./cmd

# 运行
./nofx-go
```

## 📁 项目结构

```
nofx-go/
├── cmd/                    # 主程序入口
│   ├── main.go
│   └── loadtest/           # 压力测试工具
├── internal/               # 内部包
│   ├── ai/                 # AI交易员
│   ├── bot/                # 交易机器人
│   ├── config/             # 配置管理
│   ├── exchange/           # 交易所接口
│   ├── execution/           # 执行引擎
│   ├── indicators/          # 技术指标
│   ├── metrics/             # 性能监控
│   ├── scanner/             # 市场扫描器
│   ├── strategies/          # 交易策略
│   ├── utils/               # 工具函数
│   └── web/                 # Web服务
├── pkg/                    # 公共包
│   └── types/              # 类型定义
├── tests/                  # 测试
└── README.md
```

## ⚙️ 配置

主要配置项（通过环境变量或`.env`文件）：

- `BINANCE_API_KEY`: Binance API密钥
- `BINANCE_SECRET_KEY`: Binance密钥
- `REDIS_HOST`: Redis主机
- `REDIS_PORT`: Redis端口
- `AI_PROVIDER`: AI提供商（deepseek/openai/gemini）
- `AI_MODE`: 交易模式（ai/rule）
- `DRY_RUN`: 是否启用模拟模式

完整配置项请参考 `internal/config/config.go`

## 📊 性能监控

系统自动收集以下指标：
- HTTP请求统计
- WebSocket连接统计
- 系统资源使用
- 业务指标（信号、订单、AI请求）

查看指标：
```bash
redis-cli GET "nofx:metrics:performance"
```

## 🧪 压力测试

```bash
# 使用Go测试框架
go test ./tests -run TestLoadTest -v

# 或使用独立工具
go run ./cmd/loadtest/main.go -url http://localhost:8000 -c 10 -n 1000
```

## 📝 许可证

[您的许可证]

## 🤝 贡献

欢迎提交Issue和Pull Request！

## 📧 联系方式

[您的联系方式]
