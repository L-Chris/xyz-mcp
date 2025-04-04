package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/olekukonko/tablewriter"
	"github.com/ultrazg/xyz/service"
)

var baseURL = "http://127.0.0.1:8080"

// 添加全局token变量
var (
	accessToken  string
	refreshToken string
	// token文件相关路径
	tokenDir  string
	tokenFile string
)

// 定义token结构体
type TokenConfig struct {
	AccessToken  string `json:"x-jike-access-token"`
	RefreshToken string `json:"x-jike-refresh-token"`
}

func main() {
	// 初始化token路径
	if err := initTokens(); err != nil {
		fmt.Printf("初始化token路径失败: %v\n", err)
		return
	}

	// 创建一个带取消功能的context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动前加载token
	if err := loadTokens(); err != nil {
		fmt.Printf("加载token失败: %v\n", err)
	}

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
			mcp.DefaultString("EPISODE"),
		),
		mcp.WithString("pid",
			mcp.Description("如果要在节目内搜索单集，需要传入节目的pid，并将type参数指定为EPISODE"),
			mcp.DefaultString(""),
		),
		mcp.WithString("loadMoreKey",
			mcp.Description("分页查询的条件，由接口返回"),
		),
	), handleSearch)

	// 添加刷新token工具
	s.AddTool(mcp.NewTool("refreshToken",
		mcp.WithDescription("刷新token，当接口返回401时调用此接口以获取有效的token信息"),
	), handleRefreshToken)

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
			cancel() // 触发context取消
		}
	}()

	// 在另一个 goroutine 中启动服务
	go func() {
		// 假设service.Start支持context，如果不支持需要修改service包
		err := service.Start()
		if err != nil {
			fmt.Println("Service failed:", err)
			errChan <- err
			cancel() // 触发context取消
		}
	}()

	// 等待错误或信号
	select {
	case err := <-errChan:
		fmt.Printf("程序因错误退出: %v\n", err)
		// 这里不需要调用cancel()，因为发送错误的goroutine已经调用过了
	case sig := <-sigChan:
		fmt.Printf("收到信号: %v，正在关闭...\n", sig)
		cancel() // 触发context取消
	case <-ctx.Done():
		fmt.Println("正在关闭所有服务...")
	}

	// 给清理操作留出一些时间
	time.Sleep(time.Second)
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
	var newAccessToken, newRefreshToken string
	if data, ok := result["data"].(map[string]interface{}); ok {
		newAccessToken, _ = data["x-jike-access-token"].(string)
		newRefreshToken, _ = data["x-jike-refresh-token"].(string)
	}

	// 保存令牌
	if newAccessToken != "" && newRefreshToken != "" {
		if err := saveTokens(newAccessToken, newRefreshToken); err != nil {
			return mcp.NewToolResultError("令牌保存失败: " + err.Error()), err
		}
		return mcp.NewToolResultText("登录成功！令牌已保存"), nil
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
	searchType := "EPISODE" // 设置默认值
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

	// 使用bytes.Buffer创建输出缓冲区
	var buf bytes.Buffer
	table := tablewriter.NewWriter(&buf)

	// 设置表头
	table.SetHeader([]string{"标题", "发布日期", "播放数", "评论数", "收藏数", "链接"})

	// 设置markdown格式
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")

	// 从结果中提取数据数组
	if data, ok := result["data"].(map[string]interface{}); ok {
		if items, ok := data["data"].([]interface{}); ok {
			for _, item := range items {
				if episode, ok := item.(map[string]interface{}); ok {
					// 提取标题
					title := episode["title"].(string)

					// 提取并格式化发布日期
					var pubDate string
					if pubDateStr, ok := episode["pubDate"].(string); ok {
						// 解析ISO 8601格式的时间
						if t, err := time.Parse(time.RFC3339Nano, pubDateStr); err == nil {
							pubDate = t.Format("2006/01/02")
						} else {
							pubDate = "未知日期"
						}
					}

					// 提取统计数据
					playCount := "0"
					if pc, ok := episode["playCount"].(float64); ok {
						playCount = fmt.Sprintf("%.0f", pc)
					}

					commentCount := "0"
					if cc, ok := episode["commentCount"].(float64); ok {
						commentCount = fmt.Sprintf("%.0f", cc)
					}

					favoriteCount := "0"
					if fc, ok := episode["favoriteCount"].(float64); ok {
						favoriteCount = fmt.Sprintf("%.0f", fc)
					}

					// 提取URL
					url := ""
					if media, ok := episode["media"].(map[string]interface{}); ok {
						if source, ok := media["source"].(map[string]interface{}); ok {
							if u, ok := source["url"].(string); ok {
								url = u
							}
						}
					}

					// 添加表格行
					table.Append([]string{
						title,
						pubDate,
						playCount,
						commentCount,
						favoriteCount,
						url,
					})
				}
			}
		}
	}

	// 渲染表格
	table.Render()
	return mcp.NewToolResultText(buf.String()), nil
}

// 添加新的处理函数
func handleRefreshToken(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 检查全局token
	if accessToken == "" || refreshToken == "" {
		return mcp.NewToolResultError("无效的令牌信息，请重新登录"), nil
	}

	// 发送刷新token请求
	client := resty.New()
	var result map[string]interface{}
	_, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(map[string]string{
			"x-jike-access-token":  accessToken,
			"x-jike-refresh-token": refreshToken,
		}).
		SetResult(&result).
		Post(baseURL + "/refresh_token")

	if err != nil {
		return mcp.NewToolResultError("刷新token请求失败: " + err.Error()), err
	}

	// 提取新的令牌
	var newAccessToken, newRefreshToken string
	if data, ok := result["data"].(map[string]interface{}); ok {
		newAccessToken, _ = data["x-jike-access-token"].(string)
		newRefreshToken, _ = data["x-jike-refresh-token"].(string)
	}

	// 保存新的令牌
	if newAccessToken != "" && newRefreshToken != "" {
		if err := saveTokens(newAccessToken, newRefreshToken); err != nil {
			return mcp.NewToolResultError("新令牌保存失败: " + err.Error()), err
		}
		return mcp.NewToolResultText("token刷新成功！新的令牌已保存"), nil
	}

	jsonStr, _ := json.Marshal(result)
	return mcp.NewToolResultError("刷新token失败，未能获取新的令牌: " + string(jsonStr)), nil
}

// 添加初始化路径的函数
func initTokens() error {
	// 获取用户主目录
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("获取用户主目录失败: %v", err)
	}

	// 设置完整的目录路径
	tokenDir = filepath.Join(homeDir, "mcp", "xyz-mcp")
	tokenFile = filepath.Join(tokenDir, "xyz-mcp.json")

	// 创建目录（如果不存在）
	if err := os.MkdirAll(tokenDir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %v", err)
	}

	// 检查token文件是否存在，不存在则创建
	if _, err := os.Stat(tokenFile); os.IsNotExist(err) {
		// 使用结构体创建初始化数据
		config := TokenConfig{
			AccessToken:  "",
			RefreshToken: "",
		}

		data, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			return fmt.Errorf("序列化token失败: %v", err)
		}

		if err := os.WriteFile(tokenFile, data, 0644); err != nil {
			return fmt.Errorf("创建token文件失败: %v", err)
		}
	}

	return nil
}

func saveTokens(newAccessToken, newRefreshToken string) error {
	// 更新全局变量
	accessToken = newAccessToken
	refreshToken = newRefreshToken

	// 创建token配置对象
	config := TokenConfig{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}

	// 使用MarshalIndent生成格式化的JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化token失败: %v", err)
	}

	return os.WriteFile(tokenFile, data, 0644)
}

func loadTokens() error {
	// 读取文件
	data, err := os.ReadFile(tokenFile)
	if err != nil {
		return fmt.Errorf("读取token文件失败: %v", err)
	}

	// 解析到结构体
	var config TokenConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("解析token文件失败: %v", err)
	}

	// 更新全局变量
	accessToken = config.AccessToken
	refreshToken = config.RefreshToken
	return nil
}
