package main

import (
	"context"
	"encoding/json"
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
		"小宇宙 mcp server",
		"1.0.0",
	)

	// Add tool handler
	s.AddTool(mcp.NewTool("sendCode",
		mcp.WithDescription("用于接收验证码"),
		mcp.WithString("mobilePhoneNumber",
			mcp.Required(),
			mcp.Description("手机号"),
		),
	), handleSendCode)

	s.AddTool(mcp.NewTool("login",
		mcp.WithDescription("登录"),
		mcp.WithString("mobilePhoneNumber",
			mcp.Required(),
			mcp.Description("手机号"),
		),
		mcp.WithString("verifyCode",
			mcp.Required(),
			mcp.Description("验证码"),
		),
	), handleLogin)

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

func handleSendCode(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	return mcp.NewToolResultText("发送验证码成功，请提供验证码给我完成登录"), nil
}

func handleLogin(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 定义登录参数结构体
	type LoginParams struct {
		MobilePhoneNumber string `json:"mobilePhoneNumber"`
		VerifyCode        string `json:"verifyCode"`
	}

	// 提取参数并进行类型断言
	mobilePhoneNumber, ok1 := request.Params.Arguments["mobilePhoneNumber"].(string)
	verifyCode, ok2 := request.Params.Arguments["verifyCode"].(string)
	if !ok1 || !ok2 {
		return mcp.NewToolResultError("手机号或验证码格式不正确"), nil
	}

	// 准备请求
	client := resty.New()
	var result map[string]interface{}
	_, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(LoginParams{
			MobilePhoneNumber: mobilePhoneNumber,
			VerifyCode:        verifyCode,
		}).
		SetResult(&result).
		Post(baseURL + "/login")

	if err != nil {
		return mcp.NewToolResultError("登录请求失败: " + err.Error()), err
	}

	// 提取令牌
	var accessToken, refreshToken string
	if data, ok := result["data"].(map[string]interface{}); ok {
		accessToken, _ = data["x-jike-access-token"].(string)
		refreshToken, _ = data["x-jike-refresh-token"].(string)
	}

	// 保存令牌到本地文件
	if accessToken != "" && refreshToken != "" {
		tokenData := fmt.Sprintf(`{
  "accessToken": "%s",
  "refreshToken": "%s"
}`, accessToken, refreshToken)

		err := os.WriteFile("./data/tokens.json", []byte(tokenData), 0644)
		if err != nil {
			return mcp.NewToolResultError("令牌保存失败: " + err.Error()), err
		}

		return mcp.NewToolResultText("登录成功！令牌已保存到本地 ./data/tokens.json 文件"), nil
	}
	jsonStr, _ := json.Marshal(result)
	return mcp.NewToolResultError("登录成功但未能获取令牌" + string(jsonStr)), nil
}
