//go:build go1.16
// +build go1.16

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

// MCP 协议结构
type JsonRpcReq struct {
	Jsonrpc string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type JsonRpcResp struct {
	Jsonrpc string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RpcError   `json:"error,omitempty"`
}

type RpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type AddArgs struct {
	A int `json:"a"`
	B int `json:"b"`
}

// 工具实现
func add(args AddArgs) interface{} {
	return map[string]interface{}{
		"content": []map[string]string{
			{"type": "text", "text": fmt.Sprintf("%d", args.A+args.B)},
		},
	}
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var req JsonRpcReq
		if err := json.Unmarshal(line, &req); err != nil {
			continue // 非 JSON 忽略
		}

		// 处理初始化握手请求
		if req.Method == "initialize" {
			resp := JsonRpcResp{
				Jsonrpc: "2.0",
				ID:      req.ID,
				Result: map[string]interface{}{
					"capabilities": map[string]interface{}{
						"tools": map[string]interface{}{
							"listChanged": true,
						},
					},
				},
			}
			_ = json.NewEncoder(os.Stdout).Encode(resp)
			os.Stdout.Sync()
		} else if req.Method == "$/cancelRequest" || req.Method == "textDocument/didOpen" {
			// 对于其他标准MCP消息，发送空响应
			resp := JsonRpcResp{
				Jsonrpc: "2.0",
				ID:      req.ID,
			}
			_ = json.NewEncoder(os.Stdout).Encode(resp)
			os.Stdout.Sync()
		} else if req.Method == "tools/list" {
			// 响应工具列表请求
			toolsList := []map[string]interface{}{
				{
					"name":        "add",
					"description": "Add two numbers",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"a": map[string]interface{}{"type": "integer", "description": "First number"},
							"b": map[string]interface{}{"type": "integer", "description": "Second number"},
						},
						"required": []string{"a", "b"},
					},
				},
			}
			resp := JsonRpcResp{
				Jsonrpc: "2.0",
				ID:      req.ID,
				Result: map[string]interface{}{
					"tools": toolsList,
				},
			}
			_ = json.NewEncoder(os.Stdout).Encode(resp)
			os.Stdout.Sync()
		} else if req.Method == "tools/call" {
			// 处理工具调用请求
			var addArgs AddArgs
			if paramsMap, ok := req.Params.(map[string]interface{}); ok {
				if arguments, ok := paramsMap["arguments"].(map[string]interface{}); ok {
					if a, ok := arguments["a"].(float64); ok {
						addArgs.A = int(a)
					}
					if b, ok := arguments["b"].(float64); ok {
						addArgs.B = int(b)
					}
				}
			}

			if addArgs.A != 0 || addArgs.B != 0 { // 检查是否有有效的参数
				resp := JsonRpcResp{
					Jsonrpc: "2.0",
					ID:      req.ID,
					Result:  add(addArgs),
				}
				_ = json.NewEncoder(os.Stdout).Encode(resp)
				os.Stdout.Sync()
			}
		}
	}
}
