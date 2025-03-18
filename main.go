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

	// 添加搜索工具
	s.AddTool(mcp.NewTool("search",
		mcp.WithDescription("搜索小宇宙内容"),
		mcp.WithString("keyword",
			mcp.Required(),
			mcp.Description("搜索关键词"),
		),
		mcp.WithString("type",
			mcp.Description("搜索类型，可选值：ALL（全部）、PODCAST（节目）、EPISODE（单集）、USER（用户）"),
			mcp.DefaultString("PODCAST"),
		),
		mcp.WithString("pid",
			mcp.Description("如果要在节目内搜索单集，需要传入节目的pid，并将type参数指定为EPISODE"),
			mcp.DefaultString(""),
		),
		mcp.WithString("loadMoreKey",
			mcp.Description("分页查询的条件，由接口返回"),
		),
	), handleSearch)

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

func handleSearch(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 提取参数
	keyword, ok1 := request.Params.Arguments["keyword"].(string)
	if !ok1 || keyword == "" {
		return mcp.NewToolResultError("搜索关键词不能为空"), nil
	}

	// 提取可选参数并设置默认值
	searchType := "PODCAST" // 设置默认值
	if typeArg, ok := request.Params.Arguments["type"]; ok && typeArg != nil {
		if typeStr, ok := typeArg.(string); ok {
			searchType = typeStr
		}
	}

	// 提取 pid 参数（可选）
	pid := ""
	if pidArg, ok := request.Params.Arguments["pid"]; ok && pidArg != nil {
		if pidStr, ok := pidArg.(string); ok {
			pid = pidStr
		}
	}

	// 提取 loadMoreKey 参数（可选）
	var loadMoreKeyObj map[string]interface{}
	if lmkArg, ok := request.Params.Arguments["loadMoreKey"]; ok && lmkArg != nil {
		// 如果是字符串，尝试解析为JSON对象
		if lmkStr, ok := lmkArg.(string); ok && lmkStr != "" {
			err := json.Unmarshal([]byte(lmkStr), &loadMoreKeyObj)
			if err != nil {
				// 如果解析失败，直接使用字符串
				loadMoreKeyObj = map[string]interface{}{
					"loadMoreKey": lmkStr,
				}
			}
		} else if lmkMap, ok := lmkArg.(map[string]interface{}); ok {
			// 如果已经是对象，直接使用
			loadMoreKeyObj = lmkMap
		}
	}

	// 构建请求
	client := resty.New()

	// 从文件读取令牌
	tokenData, err := os.ReadFile("./data/tokens.json")
	if err != nil {
		return mcp.NewToolResultError("读取令牌失败，请先登录: " + err.Error()), err
	}

	var tokens map[string]string
	if err := json.Unmarshal(tokenData, &tokens); err != nil {
		return mcp.NewToolResultError("解析令牌失败: " + err.Error()), err
	}

	accessToken := tokens["x-jike-access-token"]
	if accessToken == "" {
		return mcp.NewToolResultError("无效的访问令牌，请重新登录"), nil
	}

	// 构建搜索请求体
	requestBody := map[string]interface{}{
		"keyword": keyword,
		"type":    searchType,
	}

	// 如果有 pid 参数且 type 是 EPISODE，添加到请求体
	if pid != "" && searchType == "EPISODE" {
		requestBody["pid"] = pid
	}

	// 如果有 loadMoreKey 参数，添加到请求体
	if loadMoreKeyObj != nil {
		requestBody["loadMoreKey"] = loadMoreKeyObj
	}

	// 发送请求
	var result map[string]interface{}
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("x-jike-access-token", accessToken). // 修正了请求头名称
		SetBody(requestBody).
		SetResult(&result).
		Post(baseURL + "/search")

	if err != nil {
		return mcp.NewToolResultError("搜索请求失败: " + err.Error()), err
	}

	// 检查状态码
	if resp.StatusCode() != 200 {
		return mcp.NewToolResultError(fmt.Sprintf("搜索失败，状态码: %d", resp.StatusCode())), nil
	}

	// 格式化返回结果
	jsonResult, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("格式化结果失败: " + err.Error()), err
	}

	return mcp.NewToolResultText(string(jsonResult)), nil
}
