package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-resty/resty/v2"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/ultrazg/xyz/service"
)

var baseURL = "http://127.0.0.1:8080"

func main() {
	// Create MCP server
	s := server.NewMCPServer(
		"Demo 🚀",
		"1.0.0",
	)

	// Add tool
	tool := mcp.NewTool("sendCode",
		mcp.WithDescription("用于接收验证码"),
		mcp.WithString("mobilePhoneNumber",
			mcp.Required(),
			mcp.Description("手机号"),
		),
	)

	// Add tool handler
	s.AddTool(tool, helloSendCode)

	// 创建一个通道用于错误处理
	errChan := make(chan error, 2)

	// 创建一个通道用于信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 在 goroutine 中启动 stdio 服务器
	customPort := flag.Int("port", 8080, "自定义服务端口")
	flag.Parse()
	// 使用环境变量或其他方式设置 -p 参数
	os.Args = append(os.Args, "-p", fmt.Sprintf("%d", *customPort))
	go func() {
		if err := server.ServeStdio(s); err != nil {
			fmt.Printf("Server error: %v\n", err)
			errChan <- err
		}
	}()

	// 在另一个 goroutine 中启动服务
	go func() {
		err := service.Start()
		if err != nil {
			fmt.Println("Service failed:", err)
			errChan <- err
		}
	}()

	// 等待错误或信号
	select {
	case err := <-errChan:
		fmt.Printf("程序因错误退出: %v\n", err)
	case sig := <-sigChan:
		fmt.Printf("收到信号: %v，正在关闭...\n", sig)
		// 例如：service.Stop() 或其他清理函数
	}
}

func helloSendCode(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	mobilePhoneNumber, ok := request.Params.Arguments["mobilePhoneNumber"].(string)
	if !ok {
		return mcp.NewToolResultError("mobilePhoneNumber must be a string"), nil
	}
	client := resty.New()

	// 示例2: 发送登录请求
	type LoginRequest struct {
		MobilePhoneNumber string `json:"mobilePhoneNumber"`
	}

	var result map[string]interface{}
	_, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(LoginRequest{
			MobilePhoneNumber: mobilePhoneNumber,
		}).
		SetResult(&result).
		Post(baseURL + "/sendCode")
	if err != nil {
		return mcp.NewToolResultError("发送验证码失败"), err
	}

	fmt.Printf("登录响应: %+v\n", result)

	return mcp.NewToolResultText("发送验证码成功"), nil
}
