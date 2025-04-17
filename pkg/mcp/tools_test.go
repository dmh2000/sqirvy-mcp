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

func TestUnmarshalListToolsResponse(t *testing.T) {
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

	tests := []struct {
		name       string
		data       string
		wantResult *ListToolsResult
		wantID     RequestID
		wantErr    *RPCError
		parseErr   bool
	}{
		{
			name:       "valid response, string id",
			data:       `{"jsonrpc":"2.0","result":` + string(resultJSON) + `,"id":"tool-res-1"}`,
			wantResult: &sampleResult,
			wantID:     "tool-res-1",
		},
		{
			name:       "valid response, int id",
			data:       `{"jsonrpc":"2.0","result":` + string(resultJSON) + `,"id":310}`,
			wantResult: &sampleResult,
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
		},
		{
			name:     "malformed json",
			data:     `{"jsonrpc":"2.0", "result": {`,
			parseErr: true,
		},
		{
			name:     "missing result field",
			data:     `{"jsonrpc":"2.0","id":312}`,
			parseErr: true,
		},
		{
			name:     "null result field",
			data:     `{"jsonrpc":"2.0","result":null,"id":313}`,
			parseErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult, gotID, gotErr, parseErr := UnmarshalListToolsResponse([]byte(tt.data))

			if (parseErr != nil) != tt.parseErr {
				t.Fatalf("UnmarshalListToolsResponse() parseErr = %v, want parseErr %v", parseErr, tt.parseErr)
			}
			if tt.parseErr {
				return
			}

			if !reflect.DeepEqual(gotErr, tt.wantErr) {
				t.Errorf("UnmarshalListToolsResponse() gotErr = %v, want %v", gotErr, tt.wantErr)
			}
			if !reflect.DeepEqual(gotID, tt.wantID) {
				t.Errorf("UnmarshalListToolsResponse() gotID = %v, want %v", gotID, tt.wantID)
			}

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
				t.Errorf("UnmarshalListToolsResponse() gotResult JSON = \n%s\nwant JSON = \n%s", string(gotJSONIndent), string(wantJSONIndent))
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
		wantResult *CallToolResult // Compare raw messages
		wantID     RequestID
		wantErr    *RPCError
		parseErr   bool
	}{
		{
			name:       "valid response, string id",
			data:       `{"jsonrpc":"2.0","result":` + string(resultJSON) + `,"id":"tool-call-res-1"}`,
			wantResult: &sampleResult,
			wantID:     "tool-call-res-1",
		},
		{
			name:       "valid response, int id",
			data:       `{"jsonrpc":"2.0","result":` + string(resultJSON) + `,"id":410}`,
			wantResult: &sampleResult,
			wantID:     float64(410),
		},
		{
			name:       "tool error response (isError=true)",
			data:       `{"jsonrpc":"2.0","result":` + string(errorResultJSON) + `,"id":411}`,
			wantResult: &sampleErrorResult,
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
		},
		{
			name:     "malformed json",
			data:     `{"jsonrpc":"2.0", "result": {"content": [}`,
			parseErr: true,
		},
		{
			name:     "missing result field",
			data:     `{"jsonrpc":"2.0","id":413}`,
			parseErr: true,
		},
		{
			name:     "null result field",
			data:     `{"jsonrpc":"2.0","result":null,"id":414}`,
			parseErr: true,
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

			// Compare CallToolResult, focusing on the raw Content
			if gotResult == nil && tt.wantResult != nil {
				t.Errorf("UnmarshalCallToolResponse() gotResult is nil, want %v", tt.wantResult)
			} else if gotResult != nil && tt.wantResult == nil {
				t.Errorf("UnmarshalCallToolResponse() gotResult = %v, want nil", gotResult)
			} else if gotResult != nil && tt.wantResult != nil {
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
			}
		})
	}
}
