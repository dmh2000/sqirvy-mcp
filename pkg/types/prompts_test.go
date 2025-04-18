package mcp

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestMarshalListPromptsRequest(t *testing.T) {
	tests := []struct {
		name    string
		id      RequestID
		params  *ListPromptsParams
		want    string
		wantErr bool
	}{
		{
			name:   "nil params, string id",
			id:     "prompt-list-1",
			params: nil,
			want:   `{"jsonrpc":"2.0","method":"prompts/list","params":{},"id":"prompt-list-1"}`,
		},
		{
			name:   "with params, int id",
			id:     101,
			params: &ListPromptsParams{Cursor: "cursor-abc"},
			want:   `{"jsonrpc":"2.0","method":"prompts/list","params":{"cursor":"cursor-abc"},"id":101}`,
		},
		{
			name:   "empty params, int id",
			id:     102,
			params: &ListPromptsParams{},
			want:   `{"jsonrpc":"2.0","method":"prompts/list","params":{},"id":102}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MarshalListPromptsRequest(tt.id, tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalListPromptsRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				equal, err := jsonEqual(got, []byte(tt.want))
				if err != nil {
					t.Fatalf("Error comparing JSON: %v", err)
				}
				if !equal {
					t.Errorf("MarshalListPromptsRequest() got = %s, want %s", got, tt.want)
				}
			}
		})
	}
}

func TestUnmarshalListPromptsResponse(t *testing.T) {
	samplePrompt := Prompt{
		Name:        "generate_commit",
		Description: "Generate commit message",
		Arguments: []PromptArgument{
			{Name: "changes", Description: "Code changes", Required: true},
		},
	}
	sampleResult := ListPromptsResult{
		Prompts:    []Prompt{samplePrompt},
		NextCursor: "next-prompt-page",
	}
	resultJSON, _ := json.Marshal(sampleResult)

	tests := []struct {
		name       string
		data       string
		wantResult *ListPromptsResult
		wantID     RequestID
		wantErr    *RPCError
		parseErr   bool
	}{
		{
			name:       "valid response, string id",
			data:       `{"jsonrpc":"2.0","result":` + string(resultJSON) + `,"id":"prompt-res-1"}`,
			wantResult: &sampleResult,
			wantID:     "prompt-res-1",
		},
		{
			name:       "valid response, int id",
			data:       `{"jsonrpc":"2.0","result":` + string(resultJSON) + `,"id":110}`,
			wantResult: &sampleResult,
			wantID:     float64(110),
		},
		{
			name:   "rpc error response",
			data:   `{"jsonrpc":"2.0","error":{"code":-32600,"message":"Invalid Request"},"id":111}`,
			wantID: float64(111),
			wantErr: &RPCError{
				Code:    -32600,
				Message: "Invalid Request",
			},
		},
		{
			name:     "malformed json",
			data:     `{"jsonrpc":"2.0", "result":`,
			parseErr: true,
		},
		{
			name:     "missing result field",
			data:     `{"jsonrpc":"2.0","id":112}`,
			parseErr: true,
		},
		{
			name:     "null result field",
			data:     `{"jsonrpc":"2.0","result":null,"id":113}`,
			parseErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult, gotID, gotErr, parseErr := UnmarshalListPromptsResponse([]byte(tt.data))

			if (parseErr != nil) != tt.parseErr {
				t.Fatalf("UnmarshalListPromptsResponse() parseErr = %v, want parseErr %v", parseErr, tt.parseErr)
			}
			if tt.parseErr {
				return
			}

			if !reflect.DeepEqual(gotErr, tt.wantErr) {
				t.Errorf("UnmarshalListPromptsResponse() gotErr = %v, want %v", gotErr, tt.wantErr)
			}
			if !reflect.DeepEqual(gotID, tt.wantID) {
				t.Errorf("UnmarshalListPromptsResponse() gotID = %v, want %v", gotID, tt.wantID)
			}
			if !reflect.DeepEqual(gotResult, tt.wantResult) {
				t.Errorf("UnmarshalListPromptsResponse() gotResult = %+v, want %+v", gotResult, tt.wantResult)
			}
		})
	}
}

func TestMarshalGetPromptRequest(t *testing.T) {
	tests := []struct {
		name    string
		id      RequestID
		params  GetPromptParams
		want    string
		wantErr bool
	}{
		{
			name: "simple request, string id",
			id:   "prompt-get-1",
			params: GetPromptParams{
				Name: "summarize_text",
			},
			want: `{"jsonrpc":"2.0","method":"prompts/get","params":{"name":"summarize_text"},"id":"prompt-get-1"}`,
		},
		{
			name: "with arguments, int id",
			id:   201,
			params: GetPromptParams{
				Name: "summarize_text",
				Arguments: map[string]string{
					"text":   "Some long text...",
					"length": "short",
				},
			},
			want: `{"jsonrpc":"2.0","method":"prompts/get","params":{"arguments":{"length":"short","text":"Some long text..."},"name":"summarize_text"},"id":201}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MarshalGetPromptRequest(tt.id, tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalGetPromptRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				equal, err := jsonEqual(got, []byte(tt.want))
				if err != nil {
					t.Fatalf("Error comparing JSON: %v", err)
				}
				if !equal {
					t.Errorf("MarshalGetPromptRequest() got = %s, want %s", got, tt.want)
				}
			}
		})
	}
}

func TestUnmarshalGetPromptResponse(t *testing.T) {
	// Prepare sample content (as raw message)
	textContent := `{"type":"text","text":"Summarize this."}`
	sampleMessage := PromptMessage{
		Role:    RoleUser,
		Content: json.RawMessage(textContent),
	}
	sampleResult := GetPromptResult{
		Messages: []PromptMessage{sampleMessage},
	}
	resultJSON, _ := json.Marshal(sampleResult)

	tests := []struct {
		name       string
		data       string
		wantResult *GetPromptResult // Compare raw messages
		wantID     RequestID
		wantErr    *RPCError
		parseErr   bool
	}{
		{
			name:       "valid response, string id",
			data:       `{"jsonrpc":"2.0","result":` + string(resultJSON) + `,"id":"prompt-get-res-1"}`,
			wantResult: &sampleResult,
			wantID:     "prompt-get-res-1",
		},
		{
			name:       "valid response, int id",
			data:       `{"jsonrpc":"2.0","result":` + string(resultJSON) + `,"id":210}`,
			wantResult: &sampleResult,
			wantID:     float64(210),
		},
		{
			name:   "rpc error response",
			data:   `{"jsonrpc":"2.0","error":{"code":-32001,"message":"Prompt not found"},"id":211}`,
			wantID: float64(211),
			wantErr: &RPCError{
				Code:    -32001,
				Message: "Prompt not found",
			},
		},
		{
			name:     "malformed json",
			data:     `{"jsonrpc":"2.0", "result": {"messages": [}`,
			parseErr: true,
		},
		{
			name:     "missing result field",
			data:     `{"jsonrpc":"2.0","id":212}`,
			parseErr: true,
		},
		{
			name:     "null result field",
			data:     `{"jsonrpc":"2.0","result":null,"id":213}`,
			parseErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult, gotID, gotErr, parseErr := UnmarshalGetPromptResponse([]byte(tt.data))

			if (parseErr != nil) != tt.parseErr {
				t.Fatalf("UnmarshalGetPromptResponse() parseErr = %v, want parseErr %v", parseErr, tt.parseErr)
			}
			if tt.parseErr {
				return
			}

			if !reflect.DeepEqual(gotErr, tt.wantErr) {
				t.Errorf("UnmarshalGetPromptResponse() gotErr = %v, want %v", gotErr, tt.wantErr)
			}
			if !reflect.DeepEqual(gotID, tt.wantID) {
				t.Errorf("UnmarshalGetPromptResponse() gotID = %v, want %v", gotID, tt.wantID)
			}

			// Compare GetPromptResult, focusing on the raw Content within Messages
			if gotResult == nil && tt.wantResult != nil {
				t.Errorf("UnmarshalGetPromptResponse() gotResult is nil, want %v", tt.wantResult)
			} else if gotResult != nil && tt.wantResult == nil {
				t.Errorf("UnmarshalGetPromptResponse() gotResult = %v, want nil", gotResult)
			} else if gotResult != nil && tt.wantResult != nil {
				if len(gotResult.Messages) != len(tt.wantResult.Messages) {
					t.Errorf("UnmarshalGetPromptResponse() len(Messages) got = %d, want %d", len(gotResult.Messages), len(tt.wantResult.Messages))
				} else {
					for i := range gotResult.Messages {
						if gotResult.Messages[i].Role != tt.wantResult.Messages[i].Role {
							t.Errorf("UnmarshalGetPromptResponse() Messages[%d].Role got = %s, want %s", i, gotResult.Messages[i].Role, tt.wantResult.Messages[i].Role)
						}
						// Compare raw JSON bytes for content
						if !reflect.DeepEqual(gotResult.Messages[i].Content, tt.wantResult.Messages[i].Content) {
							t.Errorf("UnmarshalGetPromptResponse() Messages[%d].Content got = %s, want %s", i, gotResult.Messages[i].Content, tt.wantResult.Messages[i].Content)
						}
					}
				}
				// Compare other fields like Meta, Description if needed
				if gotResult.Description != tt.wantResult.Description {
					t.Errorf("UnmarshalGetPromptResponse() Description got = %s, want %s", gotResult.Description, tt.wantResult.Description)
				}
				if !reflect.DeepEqual(gotResult.Meta, tt.wantResult.Meta) {
					t.Errorf("UnmarshalGetPromptResponse() Meta got = %v, want %v", gotResult.Meta, tt.wantResult.Meta)
				}
			}
		})
	}
}
