package main

import (
	"fmt"
    "context"
	"github.com/ultrazg/xyz/service"
    "github.com/mark3labs/mcp-go/mcp"
    "github.com/mark3labs/mcp-go/server"
    "os"
    "os/signal"
    "syscall"
)

func main() {
    // Create MCP server
    s := server.NewMCPServer(
        "Demo 🚀",
        "1.0.0",
    )

    // Add tool
    tool := mcp.NewTool("hello_world",
        mcp.WithDescription("Say hello to someone"),
        mcp.WithString("name",
            mcp.Required(),
            mcp.Description("Name of the person to greet"),
        ),
    )

    // Add tool handler
    s.AddTool(tool, helloHandler)

    // 创建一个通道用于错误处理
    errChan := make(chan error, 2)
    
    // 创建一个通道用于信号处理
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    
    // 在 goroutine 中启动 stdio 服务器
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
        // 这里可以添加清理代码
        // 例如：service.Stop() 或其他清理函数
    }
}

func helloHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    name, ok := request.Params.Arguments["name"].(string)
    if !ok {
        return mcp.NewToolResultError("name must be a string"), nil
    }

    return mcp.NewToolResultText(fmt.Sprintf("Hello, %s!", name)), nil
}