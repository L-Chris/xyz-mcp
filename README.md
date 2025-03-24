# Xiaoyuzhou MCP Server

[English](README.md) | [中文](README.zh-CN.md)

This MCP server provides functionality to search and interact with Xiaoyuzhou (a Chinese podcast platform) content including podcasts and episodes.

## Features

- Send verification code and login to Xiaoyuzhou account
- Search Xiaoyuzhou podcast content (all, shows, episodes, users)
- Support searching episodes within a specific podcast
- Automatically save and refresh login tokens

## Components

### Tools

- **sendCode**
  - Get verification code for login
  - Input:
    - `mobilePhoneNumber` (string, required): Mobile phone number

- **login**
  - Login to Xiaoyuzhou account
  - Input:
    - `mobilePhoneNumber` (string, required): Mobile phone number
    - `verifyCode` (string, required): Verification code

- **search**
  - Search Xiaoyuzhou content
  - Input:
    - `keyword` (string, required): Search keyword
    - `type` (string, optional): Search type, available values: ALL, PODCAST, EPISODE, USER, defaults to "EPISODE"
    - `pid` (string, optional): If you want to search episodes within a podcast, you need to provide the podcast id and set type to EPISODE
    - `loadMoreKey` (string, optional): Pagination parameter, returned by the API

- **refreshToken**
  - Refresh token when API returns 401 to get valid token information

## Getting started

1. Clone the repository
2. Build the server:
   - Linux/macOS: `./build.sh`
   - Windows: `build.bat`
3. Start the server: Run the generated executable

### Usage with Desktop App

To integrate this server with a desktop app, add the following to your app's server configuration:

```json
{
  "mcpServers": {
    "xyz-mcp": {
      "command": "path/to/xiaoyuzhou/mcp/server/executable",
      "args": []
    }
  }
}
```

## Development

- Build: 
  - Linux/macOS: `./build.sh`
  - Windows: `build.bat`
- Test: `go test ./...`

## Dependencies

- [github.com/mark3labs/mcp-go](https://github.com/mark3labs/mcp-go): MCP Go SDK
- [github.com/go-resty/resty](https://github.com/go-resty/resty): HTTP client library
- [github.com/olekukonko/tablewriter](https://github.com/olekukonko/tablewriter): ASCII table generator

## Resources

- [Xiaoyuzhou Official Site](https://www.xiaoyuzhoufm.com/)

## License

This project is licensed under the MIT License. 