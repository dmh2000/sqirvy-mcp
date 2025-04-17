package mcp

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestMarshalErrorResponse(t *testing.T) {
	tests := []struct {
		name    string
		id      RequestID
		rpcErr  *RPCError
		want    string
		wantErr bool
	}{
		{
			name: "Invalid Parameters error with string ID",
			id:   "1",
			rpcErr: NewRPCError(ErrorCodeInvalidParams, "Invalid parameters", map[string]interface{}{
				"expectedSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"name": map[string]interface{}{"type": "string"},
						"age":  map[string]interface{}{"type": "integer"},
					},
					"required": []string{"name", "age"},
				},
				"receivedParams": map[string]interface{}{
					"name": 123, // Incorrect type
				},
			}),
			want: `{
				"jsonrpc": "2.0",
				"id": "1",
				"error": {
					"code": -32602,
					"message": "Invalid parameters",
					"data": {
						"expectedSchema": {
							"properties": {
								"age": { "type": "integer" },
								"name": { "type": "string" }
							},
							"required": ["name", "age"],
							"type": "object"
						},
						"receivedParams": {
							"name": 123
						}
					}
				}
			}`,
		},
		{
			name:   "Method Not Found error with string ID",
			id:     "2",
			rpcErr: NewRPCError(ErrorCodeMethodNotFound, "Method not found", map[string]interface{}{"requestedMethod": "/tools/unknownTool"}),
			want: `{
				"jsonrpc": "2.0",
				"id": "2",
				"error": {
					"code": -32601,
					"message": "Method not found",
					"data": {
						"requestedMethod": "/tools/unknownTool"
					}
				}
			}`,
		},
		{
			name:   "Internal Server Error with null ID",
			id:     nil, // Null ID
			rpcErr: NewRPCError(ErrorCodeInternalError, "Internal server error", map[string]interface{}{"details": "Unexpected null pointer exception in tool execution."}),
			want: `{
				"jsonrpc": "2.0",
				"id": null,
				"error": {
					"code": -32603,
					"message": "Internal server error",
					"data": {
						"details": "Unexpected null pointer exception in tool execution."
					}
				}
			}`,
		},
		{
			name:   "Simple error with int ID and no data",
			id:     123,
			rpcErr: NewRPCError(ErrorCodeInternalError, "Something failed", nil),
			want: `{
				"jsonrpc": "2.0",
				"id": 123,
				"error": {
					"code": -32603,
					"message": "Something failed"
				}
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MarshalErrorResponse(tt.id, tt.rpcErr)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalErrorResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				equal, err := jsonEqual(got, []byte(tt.want))
				if err != nil {
					t.Fatalf("Error comparing JSON: %v\nGot: %s\nWant: %s", err, got, tt.want)
				}
				if !equal {
					// For better diff, unmarshal both and compare interfaces
					var gotJSON, wantJSON interface{}
					_ = json.Unmarshal(got, &gotJSON)
					_ = json.Unmarshal([]byte(tt.want), &wantJSON)
					if !reflect.DeepEqual(gotJSON, wantJSON) {
						t.Errorf("MarshalErrorResponse() JSON mismatch\nGot:  %s\nWant: %s", got, tt.want)
					} else {
						// If DeepEqual passes but jsonEqual failed, it might be a subtle type issue (e.g. int vs float64)
						// Log it but don't fail the test if DeepEqual considers them equivalent after unmarshaling.
						t.Logf("MarshalErrorResponse() jsonEqual failed but reflect.DeepEqual passed after unmarshaling.\nGot:  %s\nWant: %s", got, tt.want)
					}
				}
			}
		})
	}
}

func TestUnmarshalErrorResponse(t *testing.T) {
	tests := []struct {
		name      string
		data      string
		wantError *RPCError
		wantID    RequestID
		wantErr   bool // Expect a general parsing error, not just an RPCError field
	}{
		{
			name: "Invalid Parameters error with string ID",
			data: `{
				"jsonrpc": "2.0",
				"id": "1",
				"error": {
					"code": -32602,
					"message": "Invalid parameters",
					"data": {
						"expectedSchema": {
							"type": "object",
							"properties": {
								"name": { "type": "string" },
								"age": { "type": "integer" }
							},
							"required": ["name", "age"]
						},
						"receivedParams": {
							"name": 123
						}
					}
				}
			}`,
			wantError: &RPCError{
				Code:    ErrorCodeInvalidParams,
				Message: "Invalid parameters",
				Data: map[string]interface{}{ // Expect data to be unmarshaled into map[string]interface{}
					"expectedSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"name": map[string]interface{}{"type": "string"},
							"age":  map[string]interface{}{"type": "integer"},
						},
						"required": []interface{}{"name", "age"}, // JSON arrays unmarshal to []interface{}
					},
					"receivedParams": map[string]interface{}{
						"name": float64(123), // JSON numbers unmarshal to float64
					},
				},
			},
			wantID: "1",
		},
		{
			name: "Method Not Found error with string ID",
			data: `{
				"jsonrpc": "2.0",
				"id": "2",
				"error": {
					"code": -32601,
					"message": "Method not found",
					"data": {
						"requestedMethod": "/tools/unknownTool"
					}
				}
			}`,
			wantError: &RPCError{
				Code:    ErrorCodeMethodNotFound,
				Message: "Method not found",
				Data:    map[string]interface{}{"requestedMethod": "/tools/unknownTool"},
			},
			wantID: "2",
		},
		{
			name: "Internal Server Error with null ID",
			data: `{
				"jsonrpc": "2.0",
				"id": null,
				"error": {
					"code": -32603,
					"message": "Internal server error",
					"data": {
						"details": "Unexpected null pointer exception in tool execution."
					}
				}
			}`,
			wantError: &RPCError{
				Code:    ErrorCodeInternalError,
				Message: "Internal server error",
				Data:    map[string]interface{}{"details": "Unexpected null pointer exception in tool execution."},
			},
			wantID: nil, // Expect nil for JSON null ID
		},
		{
			name: "Simple error with int ID and no data",
			data: `{
				"jsonrpc": "2.0",
				"id": 123,
				"error": {
					"code": -32603,
					"message": "Something failed"
				}
			}`,
			wantError: &RPCError{
				Code:    ErrorCodeInternalError,
				Message: "Something failed",
				Data:    nil, // Expect nil data
			},
			wantID: float64(123), // JSON numbers unmarshal to float64
		},
		{
			name: "Not an error response (valid result)",
			data: `{
				"jsonrpc": "2.0",
				"id": 456,
				"result": {"status": "ok"}
			}`,
			wantError: nil, // Expect nil error field
			wantID:    float64(456),
		},
		{
			name:    "Malformed JSON",
			data:    `{"jsonrpc": "2.0", "id": "err-malformed", "error": {`,
			wantErr: true, // Expect a general unmarshaling error
		},
		{
			name: "Missing error field (but valid JSON)",
			data: `{
				"jsonrpc": "2.0",
				"id": "err-missing"
			}`,
			wantError: nil, // Expect nil error field
			wantID:    "err-missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotError, gotID, err := UnmarshalErrorResponse([]byte(tt.data))

			if (err != nil) != tt.wantErr {
				t.Fatalf("UnmarshalErrorResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return // Don't check other fields if a general parse error was expected
			}

			// Compare IDs carefully, especially nil vs non-nil
			if !reflect.DeepEqual(gotID, tt.wantID) {
				t.Errorf("UnmarshalErrorResponse() ID got = %v (%T), want %v (%T)", gotID, gotID, tt.wantID, tt.wantID)
			}

			// Compare error structs using DeepEqual, which handles nested maps/slices
			if !reflect.DeepEqual(gotError, tt.wantError) {
				// Use JSON marshal for potentially better diff output
				gotJSON, _ := json.MarshalIndent(gotError, "", "  ")
				wantJSON, _ := json.MarshalIndent(tt.wantError, "", "  ")
				t.Errorf("UnmarshalErrorResponse() Error mismatch:\nGot:\n%s\nWant:\n%s", string(gotJSON), string(wantJSON))
			}
		})
	}
}
