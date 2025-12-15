# 性能监控、压力测试和配置优化

## 📊 性能监控

### 指标收集

系统自动收集以下性能指标：

1. **HTTP请求指标**
   - 总请求数
   - 成功/失败请求数
   - 平均延迟
   - 按路径和状态码分类的请求统计

2. **WebSocket指标**
   - 连接数
   - 消息发送成功/失败数

3. **系统指标**
   - Goroutine数量
   - 内存使用（Alloc/Sys）
   - GC次数

4. **业务指标**
   - 信号处理数（成功/失败）
   - 订单数（成功/失败）
   - AI请求数（成功/失败/延迟）

### 查看指标

指标保存在Redis中，key: `nofx:metrics:performance`

可以通过以下方式查看：
1. Web API: `/api/status`（包含部分指标）
2. Redis直接查询
3. 配置优化器会自动分析并给出建议

### 指标收集频率

默认每60秒收集一次（可通过`METRICS_GLOBAL_REFRESH_SEC`配置）

---

## 🚀 压力测试

### 使用测试工具

#### 方式1: Go测试框架

```bash
cd nofx-go
go test ./tests -run TestLoadTest -v
```

#### 方式2: 独立工具

```bash
cd nofx-go
go run ./cmd/loadtest/main.go \
  -url http://localhost:8000 \
  -c 10 \
  -n 1000 \
  -d 60s \
  -u admin \
  -p admin \
  -o result.json
```

### 参数说明

- `-url`: 服务器地址（默认: http://localhost:8000）
- `-c`: 并发数（默认: 10）
- `-n`: 总请求数（默认: 1000）
- `-d`: 测试持续时间（默认: 60s）
- `-u`: 用户名（默认: admin）
- `-p`: 密码（默认: admin）
- `-o`: 输出文件（JSON格式，可选）

### 测试结果

测试会输出：
- 总请求数、成功/失败数
- 平均延迟、P50/P95/P99延迟
- QPS（每秒请求数）
- 错误示例

---

## ⚙️ 配置优化

### 自动优化

系统每5分钟自动分析性能指标，并给出配置优化建议。

优化建议保存在Redis中，key: `nofx:config:recommendations`

### 优化规则

1. **HTTP延迟优化**
   - 如果平均延迟 > 200ms：建议增加状态缓存时间到30秒

2. **错误率优化**
   - 如果错误率 > 5%：建议增加HTTP超时时间到30秒

3. **系统资源优化**
   - 如果Goroutine > 1000：建议减少扫描并发数到5
   - 如果内存使用 > 500MB：建议减少市场快照缓存时间到60秒

4. **AI性能优化**
   - 如果AI延迟 > 5秒：建议减少AI批次大小到1
   - 如果AI错误率 > 10%：建议检查AI服务可用性

### 手动应用优化建议

优化建议不会自动应用，需要手动检查并应用：

```bash
# 查看优化建议
redis-cli GET "nofx:config:recommendations"
```

---

## 📈 性能基准

### 预期性能指标

- **HTTP请求延迟**: < 100ms (P95)
- **QPS**: > 100 req/s
- **成功率**: > 99%
- **内存使用**: < 200MB
- **Goroutine数**: < 500

### 优化目标

- 减少HTTP请求延迟
- 提高系统吞吐量
- 降低资源使用
- 提高系统稳定性

---

## 🔧 配置调整建议

### 根据实际使用调整

1. **高并发场景**
   - 增加`SCAN_CONCURRENCY`
   - 增加`AI_BATCH_SIZE`
   - 增加HTTP超时时间

2. **低延迟要求**
   - 减少缓存时间
   - 增加并发数
   - 优化数据库查询

3. **资源受限**
   - 减少并发数
   - 减少缓存时间
   - 减少批次大小

---

## 📝 使用示例

### 运行压力测试

```bash
# 基本测试
go run ./cmd/loadtest/main.go -url http://localhost:8000

# 高并发测试
go run ./cmd/loadtest/main.go -c 50 -n 5000 -d 120s

# 保存结果
go run ./cmd/loadtest/main.go -o loadtest_result.json
```

### 查看性能指标

```bash
# 通过Redis查看
redis-cli GET "nofx:metrics:performance" | jq

# 通过API查看
curl -u admin:admin http://localhost:8000/api/status
```

### 查看优化建议

```bash
redis-cli GET "nofx:config:recommendations" | jq
```

---

## ⚠️ 注意事项

1. **压力测试**
   - 不要在生产环境运行压力测试
   - 测试前确保有足够的资源
   - 监控系统资源使用情况

2. **配置优化**
   - 优化建议仅供参考
   - 需要根据实际情况调整
   - 建议逐步调整并观察效果

3. **性能监控**
   - 指标收集有轻微性能开销
   - 可以通过配置调整收集频率
   - 建议在生产环境启用

