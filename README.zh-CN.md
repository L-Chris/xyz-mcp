# 小宇宙 MCP 服务器

[English](README.md) | [中文](README.zh-CN.md)

本 MCP 服务器提供功能用于搜索和交互小宇宙（中国播客平台）内容，包括播客节目和单集内容。

## 功能特点

- 发送验证码并登录小宇宙账号
- 搜索小宇宙播客内容（全部、节目、单集、用户）
- 支持在特定节目内搜索单集
- 自动保存和刷新登录令牌

## 组件

### 工具

- **sendCode**
  - 用于接收验证码
  - 输入:
    - `mobilePhoneNumber` (字符串, 必填): 手机号

- **login**
  - 登录小宇宙账号
  - 输入:
    - `mobilePhoneNumber` (字符串, 必填): 手机号
    - `verifyCode` (字符串, 必填): 验证码

- **search**
  - 搜索小宇宙内容
  - 输入:
    - `keyword` (字符串, 必填): 搜索关键词
    - `type` (字符串, 可选): 搜索类型，可选值：ALL（全部）、PODCAST（节目）、EPISODE（单集）、USER（用户），默认为"EPISODE"
    - `pid` (字符串, 可选): 如果要在节目内搜索单集，需要传入节目的pid，并将type参数指定为EPISODE
    - `loadMoreKey` (字符串, 可选): 分页查询的条件，由接口返回

- **refreshToken**
  - 刷新token，当接口返回401时调用此接口以获取有效的token信息

## 开始使用

1. 克隆仓库
2. 构建服务器:
   - Linux/macOS: `./build.sh`
   - Windows: `build.bat`
3. 启动服务器: 运行生成的可执行文件

### 与桌面应用程序一起使用

要将此服务器与桌面应用程序集成，请将以下内容添加到应用程序的服务器配置中:

```json
{
  "mcpServers": {
    "xyz-mcp": {
      "command": "小宇宙MCP服务器可执行文件的路径",
      "args": []
    }
  }
}
```

## 开发

- 构建: 
  - Linux/macOS: `./build.sh`
  - Windows: `build.bat`
- 测试: `go test ./...`

## 依赖项

- [github.com/mark3labs/mcp-go](https://github.com/mark3labs/mcp-go): MCP Go SDK
- [github.com/go-resty/resty](https://github.com/go-resty/resty): HTTP客户端库
- [github.com/olekukonko/tablewriter](https://github.com/olekukonko/tablewriter): ASCII表格生成库

## 资源

- [小宇宙官网](https://www.xiaoyuzhoufm.com/)

## 许可证

本项目采用MIT许可证。 