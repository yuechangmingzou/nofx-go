package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/yourusername/nofx-go/tests"
)

func main() {
	var (
		baseURL      = flag.String("url", "http://localhost:8000", "服务器地址")
		concurrency  = flag.Int("c", 10, "并发数")
		totalRequests = flag.Int("n", 1000, "总请求数")
		duration     = flag.Duration("d", 60*time.Second, "测试持续时间")
		username     = flag.String("u", "admin", "用户名")
		password     = flag.String("p", "admin", "密码")
		output       = flag.String("o", "", "输出文件（JSON格式）")
	)
	flag.Parse()

	config := tests.LoadTestConfig{
		BaseURL:       *baseURL,
		Concurrency:   *concurrency,
		TotalRequests: *totalRequests,
		Duration:      *duration,
		Username:      *username,
		Password:      *password,
	}

	fmt.Printf("开始负载测试...\n")
	fmt.Printf("  服务器: %s\n", config.BaseURL)
	fmt.Printf("  并发数: %d\n", config.Concurrency)
	fmt.Printf("  总请求数: %d\n", config.TotalRequests)
	fmt.Printf("  持续时间: %v\n", config.Duration)
	fmt.Println()

	result, err := tests.RunLoadTest(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "负载测试失败: %v\n", err)
		os.Exit(1)
	}

	// 输出结果
	fmt.Println("负载测试结果:")
	fmt.Printf("  总请求数: %d\n", result.TotalRequests)
	fmt.Printf("  成功请求: %d\n", result.SuccessRequests)
	fmt.Printf("  失败请求: %d\n", result.FailedRequests)
	fmt.Printf("  成功率: %.2f%%\n", float64(result.SuccessRequests)/float64(result.TotalRequests)*100)
	fmt.Printf("  平均延迟: %v\n", result.AvgLatency)
	fmt.Printf("  最小延迟: %v\n", result.MinLatency)
	fmt.Printf("  最大延迟: %v\n", result.MaxLatency)
	fmt.Printf("  P50延迟: %v\n", result.P50Latency)
	fmt.Printf("  P95延迟: %v\n", result.P95Latency)
	fmt.Printf("  P99延迟: %v\n", result.P99Latency)
	fmt.Printf("  QPS: %.2f\n", result.RequestsPerSec)

	if len(result.Errors) > 0 {
		fmt.Printf("\n错误示例（前10个）:\n")
		for i, err := range result.Errors {
			fmt.Printf("  %d. %s\n", i+1, err)
		}
	}

	// 保存到文件
	if *output != "" {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "序列化结果失败: %v\n", err)
			os.Exit(1)
		}

		if err := os.WriteFile(*output, data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "写入文件失败: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("\n结果已保存到: %s\n", *output)
	}
}

