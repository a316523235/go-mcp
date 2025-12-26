# go-mcp
auto task

### 本地测试
```shell
## 编译
go install

## 启动
go-mcp

## 输入
{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"arguments":{"a":3,"b":4}}}

## 输出
## {"jsonrpc":"2.0","id":1,"result":{"content":[{"text":"7","type":"text"}]}}
```

### 配置到通义灵码
```json
{
  "mcpServers": {
    "go-mcp": {
      "command": "go-mcp"
    }
  }
}
```