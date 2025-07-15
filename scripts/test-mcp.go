package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

// MCP JSON-RPC structures
type MCPRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type InitializeParams struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ClientInfo      ClientInfo             `json:"clientInfo"`
}

type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ToolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

func main() {
	fmt.Println("🧪 MCP Server Test Suite")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Check if binary exists (look in parent directory)
	binaryPath := "../remember-me-mcp"
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		fmt.Println("❌ Binary not found. Run 'make build' first.")
		os.Exit(1)
	}

	// Create MCP server instance
	tester := &MCPTester{}
	
	// Run tests
	if err := tester.RunTests(); err != nil {
		fmt.Printf("❌ Test failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ All tests passed!")
	fmt.Println()
	fmt.Println("🏗️  Understanding MCP Server Architecture:")
	fmt.Println("   • MCP servers are persistent processes that communicate via stdin/stdout")
	fmt.Println("   • They use JSON-RPC 2.0 protocol")
	fmt.Println("   • Claude Desktop manages the server lifecycle")
	fmt.Println("   • Your server is working correctly!")
	fmt.Println()
	fmt.Println("📝 Next Steps:")
	fmt.Println("   1. Configure Claude Desktop: make configure-claude")
	fmt.Println("   2. Start database: make docker-db")
	fmt.Println("   3. Restart Claude Desktop")
	fmt.Println("   4. Test with natural language!")
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

type MCPTester struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	reader *bufio.Reader
}

func (t *MCPTester) RunTests() error {
	// Start MCP server
	fmt.Println("🚀 Starting MCP server...")
	if err := t.startServer(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}
	defer t.cleanup()

	// Run test sequence
	tests := []struct {
		name string
		fn   func() error
	}{
		{"Initialize connection", t.testInitialize},
		{"List tools", t.testListTools},
		{"Store memory", t.testStoreMemory},
		{"Search memories", t.testSearchMemories},
	}

	for _, test := range tests {
		fmt.Printf("🧪 %s... ", test.name)
		if err := test.fn(); err != nil {
			fmt.Printf("❌ FAILED\n")
			return fmt.Errorf("test '%s' failed: %w", test.name, err)
		}
		fmt.Printf("✅ PASSED\n")
		time.Sleep(1 * time.Second) // Delay between tests
	}

	return nil
}

func (t *MCPTester) startServer() error {
	// Don't use context with timeout for the process itself
	t.cmd = exec.Command("../remember-me-mcp")
	
	stdin, err := t.cmd.StdinPipe()
	if err != nil {
		return err
	}
	t.stdin = stdin

	stdout, err := t.cmd.StdoutPipe()
	if err != nil {
		return err
	}
	t.stdout = stdout
	t.reader = bufio.NewReader(stdout)

	// Capture stderr but don't block on it
	stderr, err := t.cmd.StderrPipe()
	if err != nil {
		return err
	}
	
	// Start the process
	if err := t.cmd.Start(); err != nil {
		return err
	}

	// Start a goroutine to consume stderr so it doesn't block
	go func() {
		stderrReader := bufio.NewReader(stderr)
		for {
			_, err := stderrReader.ReadString('\n')
			if err != nil {
				break
			}
		}
	}()

	// Give server a moment to start
	time.Sleep(3 * time.Second)

	return nil
}

func (t *MCPTester) cleanup() {
	if t.stdin != nil {
		t.stdin.Close()
	}
	if t.stdout != nil {
		t.stdout.Close()
	}
	if t.cmd != nil && t.cmd.Process != nil {
		t.cmd.Process.Kill()
		t.cmd.Wait()
	}
}

func (t *MCPTester) sendRequest(req MCPRequest) (*MCPResponse, error) {
	// Send request
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	// Debug: print what we're sending
	fmt.Printf("   Sending: %s\n", string(reqBytes))

	if _, err := t.stdin.Write(append(reqBytes, '\n')); err != nil {
		return nil, err
	}

	// Read response with timeout
	responseChan := make(chan string, 1)
	errorChan := make(chan error, 1)

	go func() {
		// Read with a larger buffer to handle long responses
		buf := make([]byte, 65536) // 64KB buffer
		n, err := t.reader.Read(buf)
		if err != nil {
			errorChan <- err
			return
		}
		line := string(buf[:n])
		// Find the end of the JSON response
		if idx := strings.Index(line, "\n"); idx != -1 {
			line = line[:idx]
		}
		responseChan <- strings.TrimSpace(line)
	}()

	select {
	case response := <-responseChan:
		// Debug: print the raw response
		fmt.Printf("   Raw response: %s\n", response)
		
		var mcpResp MCPResponse
		if err := json.Unmarshal([]byte(response), &mcpResp); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}
		return &mcpResp, nil
	case err := <-errorChan:
		return nil, fmt.Errorf("read error: %w", err)
	case <-time.After(10 * time.Second):
		return nil, fmt.Errorf("timeout waiting for response")
	}
}

func (t *MCPTester) testInitialize() error {
	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: InitializeParams{
			ProtocolVersion: "2024-11-05",
			Capabilities: map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			ClientInfo: ClientInfo{
				Name:    "test-client",
				Version: "1.0.0",
			},
		},
	}

	resp, err := t.sendRequest(req)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("initialize failed: %s", resp.Error.Message)
	}

	if resp.Result == nil {
		return fmt.Errorf("no result in initialize response")
	}

	return nil
}

func (t *MCPTester) testListTools() error {
	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/list",
		Params:  map[string]interface{}{},
	}

	resp, err := t.sendRequest(req)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("list tools failed: %s", resp.Error.Message)
	}

	// Check if we got tools in response
	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		return fmt.Errorf("unexpected result type")
	}

	tools, ok := result["tools"].([]interface{})
	if !ok {
		return fmt.Errorf("no tools array in response")
	}

	// Check for our three tools
	expectedTools := []string{"store_memory", "search_memories", "delete_memory"}
	foundTools := make(map[string]bool)
	
	for _, tool := range tools {
		if toolMap, ok := tool.(map[string]interface{}); ok {
			if name, ok := toolMap["name"].(string); ok {
				foundTools[name] = true
			}
		}
	}

	for _, expected := range expectedTools {
		if !foundTools[expected] {
			return fmt.Errorf("missing tool: %s", expected)
		}
	}

	return nil
}

func (t *MCPTester) testStoreMemory() error {
	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      3,
		Method:  "tools/call",
		Params: ToolCallParams{
			Name: "store_memory",
			Arguments: map[string]interface{}{
				"content":  "Testing memory storage from Go test",
				"type":     "fact",
				"category": "personal",
				"metadata": map[string]interface{}{
					"source": "go_test",
					"test":   true,
				},
			},
		},
	}

	resp, err := t.sendRequest(req)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("store memory failed: %s", resp.Error.Message)
	}

	// Check if we got success response
	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		return fmt.Errorf("unexpected result type")
	}

	content, ok := result["content"].([]interface{})
	if !ok || len(content) == 0 {
		return fmt.Errorf("no content in store response")
	}

	// Check if the content contains success response
	if len(content) > 0 {
		if contentItem, ok := content[0].(map[string]interface{}); ok {
			if text, ok := contentItem["text"].(string); ok {
				// Try to parse the JSON response
				var storeResponse map[string]interface{}
				if err := json.Unmarshal([]byte(text), &storeResponse); err == nil {
					if success, ok := storeResponse["success"].(bool); ok && success {
						return nil
					}
				}
			}
		}
	}

	return fmt.Errorf("store memory did not return success response")
}

func (t *MCPTester) testSearchMemories() error {
	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      4,
		Method:  "tools/call",
		Params: ToolCallParams{
			Name: "search_memories",
			Arguments: map[string]interface{}{
				"query": "test",
				"limit": 5,
			},
		},
	}

	resp, err := t.sendRequest(req)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("search memories failed: %s", resp.Error.Message)
	}

	// Check if we got success response
	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		return fmt.Errorf("unexpected result type")
	}

	content, ok := result["content"].([]interface{})
	if !ok || len(content) == 0 {
		return fmt.Errorf("no content in search response")
	}

	// Check if we found our stored memory
	contentMap, ok := content[0].(map[string]interface{})
	if !ok {
		return fmt.Errorf("unexpected content format")
	}

	// Parse the JSON text response
	text, ok := contentMap["text"].(string)
	if !ok {
		return fmt.Errorf("no text in content response")
	}

	var searchResponse map[string]interface{}
	if err := json.Unmarshal([]byte(text), &searchResponse); err != nil {
		return fmt.Errorf("failed to parse search response JSON: %w", err)
	}

	memories, ok := searchResponse["memories"].([]interface{})
	if !ok {
		return fmt.Errorf("no memories array in response")
	}

	// Should find at least one memory (the one we just stored)
	if len(memories) == 0 {
		return fmt.Errorf("no memories found in search")
	}

	// Check if one of the memories contains our test content
	found := false
	for _, mem := range memories {
		if memMap, ok := mem.(map[string]interface{}); ok {
			if content, ok := memMap["content"].(string); ok {
				if strings.Contains(content, "Testing memory storage from Go test") {
					found = true
					break
				}
			}
		}
	}

	if !found {
		return fmt.Errorf("stored memory not found in search results")
	}

	return nil
}