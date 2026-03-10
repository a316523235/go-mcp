package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	_ "github.com/go-sql-driver/mysql" // MySQL驱动，确认你的版本是否支持1.7
	"io"
	"log"
	"os"
)

// JSON-RPC 2.0 消息结构
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	ID      interface{}     `json:"id"`
}

type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
	ID      interface{} `json:"id"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// 工具定义
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

// 初始化响应结构
type InitializeResult struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ServerInfo      struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"serverInfo"`
}

// 工具列表响应
type ToolsListResult struct {
	Tools []Tool `json:"tools"`
}

// 查询参数
type QueryArgs struct {
	SQL string `json:"sql"`
}

// 列出表参数
type ListTablesArgs struct {
	Database string `json:"database"`
}

// 列出数据库参数（无参数，留空）
type ListDatabasesArgs struct {
}

func main() {
	var dsn string
	flag.StringVar(&dsn, "dsn", "", "MySQL DSN (either standard format or mysql:// URL)")
	flag.Parse()
	if dsn == "" {
		// 从环境变量读取数据库连接
		dsn = os.Getenv("dsn")
	}

	if dsn == "" {
		log.Fatal("dsn environment variable not set")
	}

	// 循环读取标准输入
	decoder := json.NewDecoder(os.Stdin)
	for {
		var req JSONRPCRequest
		if err := decoder.Decode(&req); err != nil {
			if err == io.EOF {
				break
			}
			// 忽略空行或无效输入
			continue
		}

		// 处理请求
		response := handleRequest(req, dsn)

		// 发送响应
		if err := json.NewEncoder(os.Stdout).Encode(response); err != nil {
			log.Printf("Failed to encode response: %v", err)
		}
	}
}

func handleRequest(req JSONRPCRequest, dsn string) JSONRPCResponse {
	switch req.Method {
	case "initialize":
		return handleInitialize(req)
	case "tools/list":
		return handleToolsList(req)
	case "tools/call":
		return handleToolsCall(req, dsn)
	default:
		return JSONRPCResponse{
			JSONRPC: "2.0",
			Error: &RPCError{
				Code:    -32601,
				Message: "Method not found",
			},
			ID: req.ID,
		}
	}
}

func handleInitialize(req JSONRPCRequest) JSONRPCResponse {
	// 解析初始化参数（如果需要）

	result := InitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities: map[string]interface{}{
			"tools": map[string]bool{
				"listChanged": false,
			},
		},
	}
	result.ServerInfo.Name = "mysql-mcp"
	result.ServerInfo.Version = "1.0.0"

	return JSONRPCResponse{
		JSONRPC: "2.0",
		Result:  result,
		ID:      req.ID,
	}
}

func handleToolsList(req JSONRPCRequest) JSONRPCResponse {
	tools := []Tool{
		{
			Name:        "query_mysql",
			Description: "执行SQL查询并返回结果",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"sql": map[string]string{
						"type":        "string",
						"description": "要执行的SQL语句",
					},
				},
				"required": []string{"sql"},
			},
		},
		{
			Name:        "list_databases",
			Description: "列出MySQL实例中的所有数据库",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
				"required":   []string{},
			},
		},
		{
			Name:        "list_tables",
			Description: "列出指定数据库中的所有表",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"database": map[string]string{
						"type":        "string",
						"description": "数据库名称",
					},
				},
				"required": []string{"database"},
			},
		},
	}

	return JSONRPCResponse{
		JSONRPC: "2.0",
		Result: ToolsListResult{
			Tools: tools,
		},
		ID: req.ID,
	}
}

func handleToolsCall(req JSONRPCRequest, dsn string) JSONRPCResponse {
	// 解析调用参数
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return JSONRPCResponse{
			JSONRPC: "2.0",
			Error: &RPCError{
				Code:    -32602,
				Message: "Invalid params: " + err.Error(),
			},
			ID: req.ID,
		}
	}

	// 根据工具名称分派处理
	switch params.Name {
	case "query_mysql":
		return handleQueryMySQL(req, dsn, params.Arguments)
	case "list_databases":
		return handleListDatabases(req, dsn)
	case "list_tables":
		return handleListTables(req, dsn, params.Arguments)
	default:
		return JSONRPCResponse{
			JSONRPC: "2.0",
			Error: &RPCError{
				Code:    -32601,
				Message: "Tool not found: " + params.Name,
			},
			ID: req.ID,
		}
	}
}

func handleQueryMySQL(req JSONRPCRequest, dsn string, arguments json.RawMessage) JSONRPCResponse {
	// 解析参数
	var args QueryArgs
	if err := json.Unmarshal(arguments, &args); err != nil {
		return JSONRPCResponse{
			JSONRPC: "2.0",
			Error: &RPCError{
				Code:    -32602,
				Message: "Invalid arguments: " + err.Error(),
			},
			ID: req.ID,
		}
	}

	// 使用传入的dsn建立数据库连接
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return JSONRPCResponse{
			JSONRPC: "2.0",
			Error: &RPCError{
				Code:    -32000,
				Message: "Database connection failed: " + err.Error(),
			},
			ID: req.ID,
		}
	}
	defer db.Close()

	// 执行SQL查询
	rows, err := db.Query(args.SQL)
	if err != nil {
		return JSONRPCResponse{
			JSONRPC: "2.0",
			Error: &RPCError{
				Code:    -32000,
				Message: "Query failed: " + err.Error(),
			},
			ID: req.ID,
		}
	}
	defer rows.Close()

	// 获取列名
	columns, err := rows.Columns()
	if err != nil {
		return JSONRPCResponse{
			JSONRPC: "2.0",
			Error: &RPCError{
				Code:    -32000,
				Message: "Failed to get columns: " + err.Error(),
			},
			ID: req.ID,
		}
	}

	// 构建结果
	results := make([]map[string]interface{}, 0)
	for rows.Next() {
		// 创建扫描目标
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return JSONRPCResponse{
				JSONRPC: "2.0",
				Error: &RPCError{
					Code:    -32000,
					Message: "Row scan failed: " + err.Error(),
				},
				ID: req.ID,
			}
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			// 处理 []byte 类型，转为字符串
			if b, ok := values[i].([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = values[i]
			}
		}
		results = append(results, row)
	}

	// 转换为MCP要求的格式
	resultJSON, _ := json.Marshal(results)

	response := map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": string(resultJSON),
			},
		},
	}

	return JSONRPCResponse{
		JSONRPC: "2.0",
		Result:  response,
		ID:      req.ID,
	}
}

func handleListDatabases(req JSONRPCRequest, dsn string) JSONRPCResponse {
	// 使用传入的dsn建立数据库连接
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return JSONRPCResponse{
			JSONRPC: "2.0",
			Error: &RPCError{
				Code:    -32000,
				Message: "Database connection failed: " + err.Error(),
			},
			ID: req.ID,
		}
	}
	defer db.Close()

	// 执行SHOW DATABASES查询
	rows, err := db.Query("SHOW DATABASES")
	if err != nil {
		return JSONRPCResponse{
			JSONRPC: "2.0",
			Error: &RPCError{
				Code:    -32000,
				Message: "Query failed: " + err.Error(),
			},
			ID: req.ID,
		}
	}
	defer rows.Close()

	// 获取列名
	columns, err := rows.Columns()
	if err != nil {
		return JSONRPCResponse{
			JSONRPC: "2.0",
			Error: &RPCError{
				Code:    -32000,
				Message: "Failed to get columns: " + err.Error(),
			},
			ID: req.ID,
		}
	}

	// 构建结果
	results := make([]map[string]interface{}, 0)
	for rows.Next() {
		// 创建扫描目标
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return JSONRPCResponse{
				JSONRPC: "2.0",
				Error: &RPCError{
					Code:    -32000,
					Message: "Row scan failed: " + err.Error(),
				},
				ID: req.ID,
			}
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			// 处理 []byte 类型，转为字符串
			if b, ok := values[i].([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = values[i]
			}
		}
		results = append(results, row)
	}

	// 转换为MCP要求的格式
	resultJSON, _ := json.Marshal(results)

	response := map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": string(resultJSON),
			},
		},
	}

	return JSONRPCResponse{
		JSONRPC: "2.0",
		Result:  response,
		ID:      req.ID,
	}
}

func handleListTables(req JSONRPCRequest, dsn string, arguments json.RawMessage) JSONRPCResponse {
	// 解析参数
	var args ListTablesArgs
	if err := json.Unmarshal(arguments, &args); err != nil {
		return JSONRPCResponse{
			JSONRPC: "2.0",
			Error: &RPCError{
				Code:    -32602,
				Message: "Invalid arguments: " + err.Error(),
			},
			ID: req.ID,
		}
	}

	// 使用传入的dsn建立数据库连接
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return JSONRPCResponse{
			JSONRPC: "2.0",
			Error: &RPCError{
				Code:    -32000,
				Message: "Database connection failed: " + err.Error(),
			},
			ID: req.ID,
		}
	}
	defer db.Close()

	// 执行SHOW TABLES查询
	rows, err := db.Query("SHOW TABLES FROM ?", args.Database)
	if err != nil {
		return JSONRPCResponse{
			JSONRPC: "2.0",
			Error: &RPCError{
				Code:    -32000,
				Message: "Query failed: " + err.Error(),
			},
			ID: req.ID,
		}
	}
	defer rows.Close()

	// 获取列名
	columns, err := rows.Columns()
	if err != nil {
		return JSONRPCResponse{
			JSONRPC: "2.0",
			Error: &RPCError{
				Code:    -32000,
				Message: "Failed to get columns: " + err.Error(),
			},
			ID: req.ID,
		}
	}

	// 构建结果
	results := make([]map[string]interface{}, 0)
	for rows.Next() {
		// 创建扫描目标
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return JSONRPCResponse{
				JSONRPC: "2.0",
				Error: &RPCError{
					Code:    -32000,
					Message: "Row scan failed: " + err.Error(),
				},
				ID: req.ID,
			}
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			// 处理 []byte 类型，转为字符串
			if b, ok := values[i].([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = values[i]
			}
		}
		results = append(results, row)
	}

	// 转换为MCP要求的格式
	resultJSON, _ := json.Marshal(results)

	response := map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": string(resultJSON),
			},
		},
	}

	return JSONRPCResponse{
		JSONRPC: "2.0",
		Result:  response,
		ID:      req.ID,
	}
}
