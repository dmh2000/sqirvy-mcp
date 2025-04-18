package mcp

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestMarshalInitializeRequest(t *testing.T) {
	tests := []struct {
		name    string
		id      RequestID
		params  InitializeParams
		want    string
		wantErr bool
	}{
		{
			name: "request with int id",
			id:   1,
			params: InitializeParams{
				ProtocolVersion: "2024-11-05",
				Capabilities: ClientCapabilities{
					Roots: &struct {
						ListChanged bool `json:"listChanged,omitempty"`
					}{ListChanged: true},
					Sampling: map[string]interface{}{}, // Explicitly empty map
				},
				ClientInfo: Implementation{
					Name:    "ExampleClient",
					Version: "1.0.0",
				},
			},
			// Note: Order of fields in JSON might vary, jsonEqual handles this
			want: `{
				"jsonrpc": "2.0",
				"id": 1,
				"method": "initialize",
				"params": {
					"protocolVersion": "2024-11-05",
					"capabilities": {
						"roots": {
							"listChanged": true
						}
					},
					"clientInfo": {
						"name": "ExampleClient",
						"version": "1.0.0"
					}
				}
			}`,
		},
		{
			name: "request with string id and minimal capabilities",
			id:   "init-req-abc",
			params: InitializeParams{
				ProtocolVersion: "2024-11-05",
				Capabilities:    ClientCapabilities{}, // Empty capabilities
				ClientInfo: Implementation{
					Name:    "MinimalClient",
					Version: "0.1.0",
				},
			},
			want: `{
				"jsonrpc": "2.0",
				"id": "init-req-abc",
				"method": "initialize",
				"params": {
					"protocolVersion": "2024-11-05",
					"capabilities": {},
					"clientInfo": {
						"name": "MinimalClient",
						"version": "0.1.0"
					}
				}
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MarshalInitializeRequest(tt.id, tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalInitializeRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				equal, err := jsonEqual(got, []byte(tt.want))
				if err != nil {
					t.Fatalf("Error comparing JSON: %v", err)
				}
				if !equal {
					// For better diff, unmarshal both and compare interfaces
					var gotJSON, wantJSON interface{}
					_ = json.Unmarshal(got, &gotJSON)
					_ = json.Unmarshal([]byte(tt.want), &wantJSON)
					if !reflect.DeepEqual(gotJSON, wantJSON) {
						t.Errorf("MarshalInitializeRequest() got = %s, want %s", got, tt.want)
					}
				}
			}
		})
	}
}

func TestUnmarshalInitializeResponse(t *testing.T) {
	// Sample result based on the user's example
	sampleResult := InitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities: ServerCapabilities{
			Logging: map[string]interface{}{},
			//Prompts:   &ServerCapabilitiesPrompts{ListChanged: true},
			Resources: &ServerCapabilitiesResources{ListChanged: true, Subscribe: false}, // Updated to use the new struct
			//Tools:     &ServerCapabilitiesTools{ListChanged: true},
		},
		ServerInfo: Implementation{
			Name:    "ExampleServer",
			Version: "1.0.0",
		},
	}
	resultJSON, _ := json.Marshal(sampleResult) // Assume no error marshalling test data

	tests := []struct {
		name       string
		data       string
		wantResult *InitializeResult
		wantID     RequestID
		wantErr    *RPCError
		parseErr   bool // Expect a general parsing error, not an RPCError
	}{
		{
			name:       "valid response, int id",
			data:       `{"jsonrpc":"2.0","id":1,"result":` + string(resultJSON) + `}`,
			wantResult: &sampleResult,
			wantID:     float64(1), // JSON numbers unmarshal to float64
		},
		{
			name:       "valid response, string id",
			data:       `{"jsonrpc":"2.0","id":"init-res-xyz","result":` + string(resultJSON) + `}`,
			wantResult: &sampleResult,
			wantID:     "init-res-xyz",
		},
		{
			name:   "rpc error response",
			data:   `{"jsonrpc":"2.0","error":{"code":-32000,"message":"Server error"},"id":2}`,
			wantID: float64(2),
			wantErr: &RPCError{
				Code:    -32000,
				Message: "Server error",
			},
		},
		{
			name:     "malformed json",
			data:     `{"jsonrpc":"2.0", "id": 3, "result": {`,
			parseErr: true,
		},
		{
			name:     "missing result field",
			data:     `{"jsonrpc":"2.0","id":4}`,
			parseErr: true, // Our func treats missing result as a parse error
		},
		{
			name:     "null result field",
			data:     `{"jsonrpc":"2.0","result":null,"id":5}`,
			parseErr: true, // Our func treats null result as a parse error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult, gotID, gotErr, parseErr := UnmarshalInitializeResponse([]byte(tt.data))

			if (parseErr != nil) != tt.parseErr {
				t.Fatalf("UnmarshalInitializeResponse() parseErr = %v, want parseErr %v", parseErr, tt.parseErr)
			}
			if tt.parseErr {
				return // Don't check other fields if a parse error was expected
			}

			if !reflect.DeepEqual(gotErr, tt.wantErr) {
				t.Errorf("UnmarshalInitializeResponse() gotErr = %v, want %v", gotErr, tt.wantErr)
			}
			if !reflect.DeepEqual(gotID, tt.wantID) {
				t.Errorf("UnmarshalInitializeResponse() gotID = %v, want %v", gotID, tt.wantID)
			}

			// Compare marshaled JSON of results for better accuracy with nested maps/structs
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
				t.Errorf("UnmarshalInitializeResponse() gotResult JSON = \n%s\nwant JSON = \n%s", string(gotJSONIndent), string(wantJSONIndent))
			}
		})
	}
}
