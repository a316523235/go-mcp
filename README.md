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

### 本地测试
```shell
## 编译
go install

## 测试运行
go-mcp --dsn "账户:密码@tcp(test-ad.mysql.meiyoudb.com:端口)/my_ad_activity?parseTime=true&loc=Local"              

## 输入
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"query_mysql","arguments":{"sql":"SELECT 1 as test"}}}
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"query_mysql","arguments":{"sql":"SELECT * FROM dtc_payment WHERE STATUS=3"}}}

## 输出
## {"jsonrpc":"2.0","result":{"content":[{"text":"[{\"test\":\"1\"}]","type":"text"}]},"id":3}
```