package mcp

import (
	"encoding/json"
	"io"
	"reflect"
	"sqirvy-mcp/pkg/utils"
	"testing"
)

func TestMarshalListResourcesRequest(t *testing.T) {
	tests := []struct {
		name    string
		id      RequestID
		params  *ListResourcesParams
		want    string
		wantErr bool
	}{
		{
			name:   "nil params, string id",
			id:     "req-1",
			params: nil,
			want:   `{"jsonrpc":"2.0","method":"resources/list","params":{},"id":"req-1"}`,
		},
		{
			name:   "with params, int id",
			id:     2,
			params: &ListResourcesParams{Cursor: "page-token-123"},
			want:   `{"jsonrpc":"2.0","method":"resources/list","params":{"cursor":"page-token-123"},"id":2}`,
		},
		{
			name:   "empty params, int id",
			id:     3,
			params: &ListResourcesParams{},
			want:   `{"jsonrpc":"2.0","method":"resources/list","params":{},"id":3}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MarshalListResourcesRequest(tt.id, tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalListResourcesRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				equal, err := jsonEqual(got, []byte(tt.want))
				if err != nil {
					t.Fatalf("Error comparing JSON: %v", err)
				}
				if !equal {
					t.Errorf("MarshalListResourcesRequest() got = %s, want %s", got, tt.want)
				}
			}
		})
	}
}

func TestMarshalListResourcesTemplatesRequest(t *testing.T) {
	tests := []struct {
		name    string
		id      RequestID
		params  *ListResourcesTemplatesParams
		want    string
		wantErr bool
	}{
		{
			name:   "nil params, string id",
			id:     "tmpl-list-1",
			params: nil,
			want:   `{"jsonrpc":"2.0","method":"resources/templates/list","params":{},"id":"tmpl-list-1"}`,
		},
		{
			name:   "with params, int id",
			id:     601,
			params: &ListResourcesTemplatesParams{Cursor: "tmpl-cursor-xyz"},
			want:   `{"jsonrpc":"2.0","method":"resources/templates/list","params":{"cursor":"tmpl-cursor-xyz"},"id":601}`,
		},
		{
			name:   "empty params, int id",
			id:     602,
			params: &ListResourcesTemplatesParams{},
			want:   `{"jsonrpc":"2.0","method":"resources/templates/list","params":{},"id":602}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MarshalListResourcesTemplatesRequest(tt.id, tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalListResourcesTemplatesRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				equal, err := jsonEqual(got, []byte(tt.want))
				if err != nil {
					t.Fatalf("Error comparing JSON: %v", err)
				}
				if !equal {
					t.Errorf("MarshalListResourcesTemplatesRequest() got = %s, want %s", got, tt.want)
				}
			}
		})
	}
}

func TestUnmarshalListResourcesTemplatesResponse(t *testing.T) {
	sampleTemplate := ResourcesTemplates{
		Name:        "random_data",
		URITemplate: "data://random_data?{length}",
		Description: "Generates random data",
		MimeType:    "text/plain",
	}
	sampleResult := ListResourcesTemplatesResult{
		ResourcesTemplates: []ResourcesTemplates{sampleTemplate},
		NextCursor:         "next-tmpl-page",
	}
	resultJSON, _ := json.Marshal(sampleResult)

	tests := []struct {
		name       string
		data       string
		wantResult *ListResourcesTemplatesResult
		wantID     RequestID
		wantErr    *RPCError
		parseErr   bool
	}{
		{
			name:       "valid response, string id",
			data:       `{"jsonrpc":"2.0","result":` + string(resultJSON) + `,"id":"tmpl-res-1"}`,
			wantResult: &sampleResult,
			wantID:     "tmpl-res-1",
		},
		{
			name:       "valid response, int id",
			data:       `{"jsonrpc":"2.0","result":` + string(resultJSON) + `,"id":610}`,
			wantResult: &sampleResult,
			wantID:     float64(610),
		},
		{
			name:   "rpc error response",
			data:   `{"jsonrpc":"2.0","error":{"code":-32600,"message":"Invalid Request"},"id":611}`,
			wantID: float64(611),
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
			data:     `{"jsonrpc":"2.0","id":612}`,
			parseErr: true,
		},
		{
			name:     "null result field",
			data:     `{"jsonrpc":"2.0","result":null,"id":613}`,
			parseErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult, gotID, gotErr, parseErr := UnmarshalListResourcesTemplatesResult([]byte(tt.data))

			if (parseErr != nil) != tt.parseErr {
				t.Fatalf("UnmarshalListResourcesTemplatesResultrseErr = %v, want parseErr %v", parseErr, tt.parseErr)
			}
			if tt.parseErr {
				return
			}

			if !reflect.DeepEqual(gotErr, tt.wantErr) {
				t.Errorf("UnmarshalListResourcesTemplatesResponse() gotErr = %v, want %v", gotErr, tt.wantErr)
			}
			if !reflect.DeepEqual(gotID, tt.wantID) {
				t.Errorf("UnmarshalListResourcesTemplatesResponse() gotID = %v, want %v", gotID, tt.wantID)
			}
			if !reflect.DeepEqual(gotResult, tt.wantResult) {
				t.Errorf("UnmarshalListResourcesTemplatesResponse() gotResult = %+v, want %+v", gotResult, tt.wantResult)
			}
		})
	}
}

func TestUnmarshalListResourcesRequest(t *testing.T) {
	// Create a test logger that discards output
	testLogger := utils.New(io.Discard, "", 0, "DEBUG")

	tests := []struct {
		name       string
		payload    string
		wantParams *ListResourcesParams
		wantID     RequestID
		wantRPCErr bool
		wantErr    bool
	}{
		{
			name:       "valid request with empty params",
			payload:    `{"jsonrpc":"2.0","method":"resources/list","params":{},"id":"test1"}`,
			wantParams: &ListResourcesParams{},
			wantID:     "test1",
		},
		{
			name:       "valid request with cursor",
			payload:    `{"jsonrpc":"2.0","method":"resources/list","params":{"cursor":"next-page-token"},"id":42}`,
			wantParams: &ListResourcesParams{Cursor: "next-page-token"},
			wantID:     float64(42),
		},
		{
			name:       "valid request with null params",
			payload:    `{"jsonrpc":"2.0","method":"resources/list","params":null,"id":"test2"}`,
			wantParams: &ListResourcesParams{},
			wantID:     "test2",
		},
		{
			name:       "invalid json",
			payload:    `{"jsonrpc":"2.0","method":"resources/list","params":{},"id":`,
			wantRPCErr: true,
			wantErr:    true,
		},
		{
			name:       "wrong method",
			payload:    `{"jsonrpc":"2.0","method":"wrong/method","params":{},"id":"test3"}`,
			wantRPCErr: true,
			wantErr:    true,
			wantID:     "test3",
		},
		{
			name:       "wrong jsonrpc version",
			payload:    `{"jsonrpc":"1.0","method":"resources/list","params":{},"id":"test4"}`,
			wantRPCErr: true,
			wantErr:    true,
			wantID:     "test4",
		},
		{
			name:       "invalid params type",
			payload:    `{"jsonrpc":"2.0","method":"resources/list","params":"invalid","id":"test5"}`,
			wantRPCErr: true,
			wantErr:    true,
			wantID:     "test5",
		},
		{
			name:       "invalid params structure",
			payload:    `{"jsonrpc":"2.0","method":"resources/list","params":{"invalid":123},"id":"test6"}`,
			wantParams: &ListResourcesParams{},
			wantID:     "test6",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotParams, gotID, gotRPCErr, gotErr := UnmarshalListResourcesRequest([]byte(tt.payload), testLogger)

			// Check error conditions
			if (gotErr != nil) != tt.wantErr {
				t.Errorf("UnmarshalListResourcesRequest() error = %v, wantErr %v", gotErr, tt.wantErr)
				return
			}
			if (gotRPCErr != nil) != tt.wantRPCErr {
				t.Errorf("UnmarshalListResourcesRequest() rpcErr = %v, wantRPCErr %v", gotRPCErr, tt.wantRPCErr)
				return
			}

			// If we expect errors, don't check the other returns
			if tt.wantErr || tt.wantRPCErr {
				// But do check ID if specified
				if tt.wantID != nil && !reflect.DeepEqual(gotID, tt.wantID) {
					t.Errorf("UnmarshalListResourcesRequest() gotID = %v, want %v", gotID, tt.wantID)
				}
				return
			}

			// Check returned values
			if !reflect.DeepEqual(gotParams, tt.wantParams) {
				t.Errorf("UnmarshalListResourcesRequest() gotParams = %v, want %v", gotParams, tt.wantParams)
			}
			if !reflect.DeepEqual(gotID, tt.wantID) {
				t.Errorf("UnmarshalListResourcesRequest() gotID = %v, want %v", gotID, tt.wantID)
			}
		})
	}
}

func TestUnmarshalListResourcesResponse(t *testing.T) {
	sampleResource := Resource{
		Name: "app.log",
		URI:  "file:///logs/app.log",
	}
	sampleResult := ListResourcesResult{
		Resources:  []Resource{sampleResource},
		NextCursor: "next-page",
	}
	resultJSON, _ := json.Marshal(sampleResult) // Assume no error marshalling test data

	tests := []struct {
		name       string
		data       string
		wantResult *ListResourcesResult
		wantID     RequestID
		wantErr    *RPCError
		parseErr   bool // Expect a general parsing error, not an RPCError
	}{
		{
			name:       "valid response, string id",
			data:       `{"jsonrpc":"2.0","result":` + string(resultJSON) + `,"id":"res-1"}`,
			wantResult: &sampleResult,
			wantID:     "res-1",
		},
		{
			name:       "valid response, int id",
			data:       `{"jsonrpc":"2.0","result":` + string(resultJSON) + `,"id":10}`,
			wantResult: &sampleResult,
			wantID:     float64(10), // JSON numbers unmarshal to float64 by default
		},
		{
			name:   "rpc error response",
			data:   `{"jsonrpc":"2.0","error":{"code":-32601,"message":"Method not found"},"id":11}`,
			wantID: float64(11),
			wantErr: &RPCError{
				Code:    -32601,
				Message: "Method not found",
			},
		},
		{
			name:     "malformed json",
			data:     `{"jsonrpc":"2.0",`,
			parseErr: true,
		},
		{
			name:     "missing result field",
			data:     `{"jsonrpc":"2.0","id":12}`,
			parseErr: true, // Our func treats missing result as a parse error
		},
		{
			name:     "null result field",
			data:     `{"jsonrpc":"2.0","result":null,"id":13}`,
			parseErr: true, // Our func treats null result as a parse error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult, gotID, gotErr, parseErr := UnmarshalListResourcesResult([]byte(tt.data))

			if (parseErr != nil) != tt.parseErr {
				t.Fatalf("UnmarshalListResourcesResponse() parseErr = %v, want parseErr %v", parseErr, tt.parseErr)
			}
			if tt.parseErr {
				return // Don't check other fields if a parse error was expected
			}

			if !reflect.DeepEqual(gotErr, tt.wantErr) {
				t.Errorf("UnmarshalListResourcesResponse() gotErr = %v, want %v", gotErr, tt.wantErr)
			}
			if !reflect.DeepEqual(gotID, tt.wantID) {
				t.Errorf("UnmarshalListResourcesResponse() gotID = %v, want %v", gotID, tt.wantID)
			}
			if !reflect.DeepEqual(gotResult, tt.wantResult) {
				t.Errorf("UnmarshalListResourcesResponse() gotResult = %v, want %v", gotResult, tt.wantResult)
			}
		})
	}
}

func TestMarshalReadResourceRequest(t *testing.T) {
	tests := []struct {
		name    string
		id      RequestID
		params  ReadResourceParams
		want    string
		wantErr bool
	}{
		{
			name:   "simple request, string id",
			id:     "req-read-1",
			params: ReadResourceParams{URI: "file:///path/to/file.txt"},
			want:   `{"jsonrpc":"2.0","method":"resources/read","params":{"uri":"file:///path/to/file.txt"},"id":"req-read-1"}`,
		},
		{
			name:   "simple request, int id",
			id:     50,
			params: ReadResourceParams{URI: "mcp://server/resource/id"},
			want:   `{"jsonrpc":"2.0","method":"resources/read","params":{"uri":"mcp://server/resource/id"},"id":50}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MarshalReadResourcesRequest(tt.id, tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalReadResourceRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				equal, err := jsonEqual(got, []byte(tt.want))
				if err != nil {
					t.Fatalf("Error comparing JSON: %v", err)
				}
				if !equal {
					t.Errorf("MarshalReadResourceRequest() got = %s, want %s", got, tt.want)
				}
			}
		})
	}
}

func TestUnmarshalReadResourceResponse(t *testing.T) {
	// Prepare sample contents
	textContent := TextResourceContents{
		URI:      "file:///logs/app.log",
		MimeType: "text/plain",
		Text:     "Log line 1",
	}
	blobContent := BlobResourceContents{
		URI:      "file:///images/logo.png",
		MimeType: "image/png",
		Blob:     "base64encodeddata", // Placeholder
	}
	textContentJSON, _ := json.Marshal(textContent)
	blobContentJSON, _ := json.Marshal(blobContent)

	// Prepare sample result with raw messages
	sampleResult := ReadResourceResult{
		Contents: []json.RawMessage{
			json.RawMessage(textContentJSON),
			json.RawMessage(blobContentJSON),
		},
	}
	resultJSON, _ := json.Marshal(sampleResult)

	tests := []struct {
		name       string
		data       string
		wantResult *ReadResourceResult // Compare raw messages
		wantID     RequestID
		wantErr    *RPCError
		parseErr   bool
	}{
		{
			name:       "valid response, string id",
			data:       `{"jsonrpc":"2.0","result":` + string(resultJSON) + `,"id":"res-read-1"}`,
			wantResult: &sampleResult,
			wantID:     "res-read-1",
		},
		{
			name:       "valid response, int id",
			data:       `{"jsonrpc":"2.0","result":` + string(resultJSON) + `,"id":51}`,
			wantResult: &sampleResult,
			wantID:     float64(51),
		},
		{
			name:   "rpc error response",
			data:   `{"jsonrpc":"2.0","error":{"code":-32000,"message":"Resource not found"},"id":52}`,
			wantID: float64(52),
			wantErr: &RPCError{
				Code:    -32000,
				Message: "Resource not found",
			},
		},
		{
			name:     "malformed json",
			data:     `{"jsonrpc":"2.0", "result": {`,
			parseErr: true,
		},
		{
			name:     "missing result field",
			data:     `{"jsonrpc":"2.0","id":53}`,
			parseErr: true,
		},
		{
			name:     "null result field",
			data:     `{"jsonrpc":"2.0","result":null,"id":54}`,
			parseErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult, gotID, gotErr, parseErr := UnmarshalReadResourcesResult([]byte(tt.data))

			if (parseErr != nil) != tt.parseErr {
				t.Fatalf("UnmarshalReadResourceResponse() parseErr = %v, want parseErr %v", parseErr, tt.parseErr)
			}
			if tt.parseErr {
				return
			}

			if !reflect.DeepEqual(gotErr, tt.wantErr) {
				t.Errorf("UnmarshalReadResourceResponse() gotErr = %v, want %v", gotErr, tt.wantErr)
			}
			if !reflect.DeepEqual(gotID, tt.wantID) {
				t.Errorf("UnmarshalReadResourceResponse() gotID = %v, want %v", gotID, tt.wantID)
			}

			// Compare ReadResourceResult, focusing on the raw Contents
			if gotResult == nil && tt.wantResult != nil {
				t.Errorf("UnmarshalReadResourceResponse() gotResult is nil, want %v", tt.wantResult)
			} else if gotResult != nil && tt.wantResult == nil {
				t.Errorf("UnmarshalReadResourceResponse() gotResult = %v, want nil", gotResult)
			} else if gotResult != nil && tt.wantResult != nil {
				if len(gotResult.Contents) != len(tt.wantResult.Contents) {
					t.Errorf("UnmarshalReadResourceResponse() len(Contents) got = %d, want %d", len(gotResult.Contents), len(tt.wantResult.Contents))
				} else {
					for i := range gotResult.Contents {
						// Compare raw JSON bytes for contents
						if !reflect.DeepEqual(gotResult.Contents[i], tt.wantResult.Contents[i]) {
							t.Errorf("UnmarshalReadResourceResponse() Contents[%d] got = %s, want %s", i, gotResult.Contents[i], tt.wantResult.Contents[i])
						}
					}
				}
				// Compare Meta if needed (currently nil in tests)
				if !reflect.DeepEqual(gotResult.Meta, tt.wantResult.Meta) {
					t.Errorf("UnmarshalReadResourceResponse() Meta got = %v, want %v", gotResult.Meta, tt.wantResult.Meta)
				}
			}
		})
	}
}
