package mcp

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestMarshalListToolsRequest(t *testing.T) {
	tests := []struct {
		name    string
		id      RequestID
		params  *ListToolsParams
		want    string
		wantErr bool
	}{
		{
			name:   "nil params, string id",
			id:     "tool-list-1",
			params: nil,
			want:   `{"jsonrpc":"2.0","method":"tools/list","params":{},"id":"tool-list-1"}`,
		},
		{
			name:   "empty params, int id",
			id:     302,
			params: &ListToolsParams{},
			want:   `{"jsonrpc":"2.0","method":"tools/list","params":{},"id":302}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MarshalListToolsRequest(tt.id, tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalListToolsRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				equal, err := jsonEqual(got, []byte(tt.want))
				if err != nil {
					t.Fatalf("Error comparing JSON: %v", err)
				}
				if !equal {
					t.Errorf("MarshalListToolsRequest() got = %s, want %s", got, tt.want)
				}
			}
		})
	}
}

func TestUnmarshalListToolsResult(t *testing.T) {
	sampleTool := Tool{
		Name:        "calculate_sum",
		Description: "Adds two numbers.",
		InputSchema: ToolInputSchema{
			"type": "object",
			"properties": map[string]interface{}{
				// Use map[string]interface{} to match unmarshaling behavior
				"a": map[string]interface{}{"type": "number"},
				"b": map[string]interface{}{"type": "number"},
			},
			"required": []string{"a", "b"},
		},
	}
	sampleResult := ListToolsResult{
		Tools: []Tool{sampleTool},
		// NextCursor removed
	}
	resultJSON, _ := json.Marshal(sampleResult)

	// Define the zero value for ListToolsResult to use in error cases
	var zeroListToolsResult ListToolsResult

	tests := []struct {
		name       string
		data       string
		wantResult ListToolsResult // Value type
		wantID     RequestID
		wantErr    *RPCError
		parseErr   bool
	}{
		{
			name:       "valid response, string id",
			data:       `{"jsonrpc":"2.0","result":` + string(resultJSON) + `,"id":"tool-res-1"}`,
			wantResult: sampleResult, // Use value
			wantID:     "tool-res-1",
		},
		{
			name:       "valid response, int id",
			data:       `{"jsonrpc":"2.0","result":` + string(resultJSON) + `,"id":310}`,
			wantResult: sampleResult, // Use value
			wantID:     float64(310),
		},
		{
			name:   "rpc error response",
			data:   `{"jsonrpc":"2.0","error":{"code":-32602,"message":"Invalid params"},"id":311}`,
			wantID: float64(311),
			wantErr: &RPCError{
				Code:    -32602,
				Message: "Invalid params",
			},
			wantResult: zeroListToolsResult, // Expect zero value on RPC error
		},
		{
			name:       "malformed json",
			data:       `{"jsonrpc":"2.0", "result": {`,
			parseErr:   true,
			wantResult: zeroListToolsResult, // Expect zero value on parse error
		},
		{
			name:       "missing result field",
			data:       `{"jsonrpc":"2.0","id":312}`,
			parseErr:   true,
			wantResult: zeroListToolsResult, // Expect zero value on parse error
		},
		{
			name:       "null result field",
			data:       `{"jsonrpc":"2.0","result":null,"id":313}`,
			parseErr:   true,
			wantResult: zeroListToolsResult, // Expect zero value on parse error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult, gotID, gotErr, parseErr := UnmarshalListToolsResult([]byte(tt.data))

			if (parseErr != nil) != tt.parseErr {
				t.Fatalf("UnmarshalListToolsResult() parseErr = %v, want parseErr %v", parseErr, tt.parseErr)
			}
			if tt.parseErr {
				return
			}

			if !reflect.DeepEqual(gotErr, tt.wantErr) {
				t.Errorf("UnmarshalListToolsResult() gotErr = %v, want %v", gotErr, tt.wantErr)
			}
			if !reflect.DeepEqual(gotID, tt.wantID) {
				t.Errorf("UnmarshalListToolsResult() gotID = %v, want %v", gotID, tt.wantID)
			}

			// Only compare results if no error was expected
			if tt.wantErr == nil && !tt.parseErr {
				// Compare marshaled JSON of results instead of DeepEqual on structs
				// due to potential type inconsistencies in nested maps after unmarshaling.
				gotJSON, err := json.Marshal(gotResult)
				if err != nil {
					t.Fatalf("Failed to marshal gotResult: %v", err)
				}
				wantJSON, err := json.Marshal(tt.wantResult)
				if err != nil {
					t.Fatalf("Failed to marshal wantResult: %v", err)
				}

				equal, err := jsonEqual(gotJSON, wantJSON)
				if err != nil {
					t.Fatalf("Error comparing result JSON: %v", err)
				}
				if !equal {
					// Indent for readability in error message
					gotJSONIndent, _ := json.MarshalIndent(gotResult, "", "  ")
					wantJSONIndent, _ := json.MarshalIndent(tt.wantResult, "", "  ")
					t.Errorf("UnmarshalListToolsResult() gotResult JSON = \n%s\nwant JSON = \n%s", string(gotJSONIndent), string(wantJSONIndent))
				}
			} else {
				// If an error was expected, ensure the returned result is the zero value
				if !reflect.DeepEqual(gotResult, zeroListToolsResult) {
					t.Errorf("UnmarshalListToolsResult() expected zero result on error, but got = %+v", gotResult)
				}
			}
		})
	}
}

func TestMarshalCallToolRequest(t *testing.T) {
	tests := []struct {
		name    string
		id      RequestID
		params  CallToolParams
		want    string
		wantErr bool
	}{
		{
			name: "simple request, string id",
			id:   "tool-call-1",
			params: CallToolParams{
				Name: "calculate_sum",
				Arguments: map[string]interface{}{
					"a": 10,
					"b": 15.5,
				},
			},
			// Note: JSON marshaling order of map keys is not guaranteed
			want: `{"jsonrpc":"2.0","method":"tools/call","params":{"arguments":{"a":10,"b":15.5},"name":"calculate_sum"},"id":"tool-call-1"}`,
		},
		{
			name: "no arguments, int id",
			id:   401,
			params: CallToolParams{
				Name: "get_time",
			},
			want: `{"jsonrpc":"2.0","method":"tools/call","params":{"name":"get_time"},"id":401}`,
		},
		{
			name: "complex arguments, int id",
			id:   402,
			params: CallToolParams{
				Name: "process_data",
				Arguments: map[string]interface{}{
					"data":   []int{1, 2, 3},
					"config": map[string]bool{"verbose": true},
				},
			},
			want: `{"jsonrpc":"2.0","method":"tools/call","params":{"arguments":{"config":{"verbose":true},"data":[1,2,3]},"name":"process_data"},"id":402}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MarshalCallToolRequest(tt.id, tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalCallToolRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				equal, err := jsonEqual(got, []byte(tt.want))
				if err != nil {
					t.Fatalf("Error comparing JSON: %v", err)
				}
				if !equal {
					// Marshal the expected string back to interface for comparison if needed
					// This helps ignore key order differences in maps
					var wantJSON, gotJSON interface{}
					_ = json.Unmarshal([]byte(tt.want), &wantJSON)
					_ = json.Unmarshal(got, &gotJSON)
					if !reflect.DeepEqual(gotJSON, wantJSON) {
						t.Errorf("MarshalCallToolRequest() got = %s, want %s", got, tt.want)
					}
				}
			}
		})
	}
}

func TestUnmarshalCallToolResponse(t *testing.T) {
	// Prepare sample content (as raw message)
	textContent := `{"type":"text","text":"Result is 25"}`
	sampleContent := []json.RawMessage{
		json.RawMessage(textContent),
	}
	sampleResult := CallToolResult{
		Content: sampleContent,
		IsError: false,
	}
	resultJSON, _ := json.Marshal(sampleResult)

	errorContent := `{"type":"text","text":"Error: Division by zero"}`
	sampleErrorContent := []json.RawMessage{
		json.RawMessage(errorContent),
	}
	sampleErrorResult := CallToolResult{
		Content: sampleErrorContent,
		IsError: true,
	}
	errorResultJSON, _ := json.Marshal(sampleErrorResult)

	tests := []struct {
		name       string
		data       string
		wantResult CallToolResult // Changed to value type
		wantID     RequestID
		wantErr    *RPCError
		parseErr   bool
	}{
		{
			name:       "valid response, string id",
			data:       `{"jsonrpc":"2.0","result":` + string(resultJSON) + `,"id":"tool-call-res-1"}`,
			wantResult: sampleResult, // Use value
			wantID:     "tool-call-res-1",
		},
		{
			name:       "valid response, int id",
			data:       `{"jsonrpc":"2.0","result":` + string(resultJSON) + `,"id":410}`,
			wantResult: sampleResult, // Use value
			wantID:     float64(410),
		},
		{
			name:       "tool error response (isError=true)",
			data:       `{"jsonrpc":"2.0","result":` + string(errorResultJSON) + `,"id":411}`,
			wantResult: sampleErrorResult, // Use value
			wantID:     float64(411),
		},
		{
			name:   "rpc error response",
			data:   `{"jsonrpc":"2.0","error":{"code":-32002,"message":"Tool execution failed"},"id":412}`,
			wantID: float64(412),
			wantErr: &RPCError{
				Code:    -32002,
				Message: "Tool execution failed",
			},
			wantResult: CallToolResult{}, // Expect zero value on RPC error
		},
		{
			name:       "malformed json",
			data:       `{"jsonrpc":"2.0", "result": {"content": [}`,
			parseErr:   true,
			wantResult: CallToolResult{}, // Expect zero value on parse error
		},
		{
			name:       "missing result field",
			data:       `{"jsonrpc":"2.0","id":413}`,
			parseErr:   true,
			wantResult: CallToolResult{}, // Expect zero value on parse error
		},
		{
			name:       "null result field",
			data:       `{"jsonrpc":"2.0","result":null,"id":414}`,
			parseErr:   true,
			wantResult: CallToolResult{}, // Expect zero value on parse error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult, gotID, gotErr, parseErr := UnmarshalCallToolResponse([]byte(tt.data))

			if (parseErr != nil) != tt.parseErr {
				t.Fatalf("UnmarshalCallToolResponse() parseErr = %v, want parseErr %v", parseErr, tt.parseErr)
			}
			if tt.parseErr {
				return
			}

			if !reflect.DeepEqual(gotErr, tt.wantErr) {
				t.Errorf("UnmarshalCallToolResponse() gotErr = %v, want %v", gotErr, tt.wantErr)
			}
			if !reflect.DeepEqual(gotID, tt.wantID) {
				t.Errorf("UnmarshalCallToolResponse() gotID = %v, want %v", gotID, tt.wantID)
			}

			// Only compare results if no error was expected
			if tt.wantErr == nil && !tt.parseErr {
				// Compare CallToolResult, focusing on the raw Content
				if gotResult.IsError != tt.wantResult.IsError {
					t.Errorf("UnmarshalCallToolResponse() IsError got = %v, want %v", gotResult.IsError, tt.wantResult.IsError)
				}
				if len(gotResult.Content) != len(tt.wantResult.Content) {
					t.Errorf("UnmarshalCallToolResponse() len(Content) got = %d, want %d", len(gotResult.Content), len(tt.wantResult.Content))
				} else {
					for i := range gotResult.Content {
						// Compare raw JSON bytes for content
						equal, err := jsonEqual(gotResult.Content[i], tt.wantResult.Content[i])
						if err != nil {
							t.Fatalf("Error comparing content JSON: %v", err)
						}
						if !equal {
							t.Errorf("UnmarshalCallToolResponse() Content[%d] got = %s, want %s", i, gotResult.Content[i], tt.wantResult.Content[i])
						}
					}
				}
				// Compare Meta if needed
				if !reflect.DeepEqual(gotResult.Meta, tt.wantResult.Meta) {
					t.Errorf("UnmarshalCallToolResponse() Meta got = %v, want %v", gotResult.Meta, tt.wantResult.Meta)
				}
			} else {
				// If an error was expected, ensure the returned result is the zero value
				var zeroResult CallToolResult
				if !reflect.DeepEqual(gotResult, zeroResult) {
					t.Errorf("UnmarshalCallToolResponse() expected zero result on error, but got = %+v", gotResult)
				}
			}
		})
	}
}
