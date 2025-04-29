in pkg/mcp/prompts.go, review the code for bugs and improvements and update the code.
all returns should be by value, not pointer reference
- CLIENT
- These functions are used by an mcp-client:
  - MarshalListPromptsRequest
  - UnmarshalListPromptsResult
  - MarshalGetPromptRequest
  - UnmarshalGetPromptResult

  - SERVER
  - all server functions should accept a "logger *utils.Logger" as a parameter. if its not there, add it. 
  - if the logger value is nil, return an error
- These functions are used by the mcp-server:
  - MarshalGetPromptResult
  - NewGetPromptResult
  - MarshalListPromptResult
  - NewListPromptResult
- add this function that isn't implemented yet
  - UnmarshalListPromptsRequest
  - UnmarshalGetPromptRequest


in pkg/mcp/tools.go, review the code for bugs and improvements and update the code.
all returns should be by value, not pointer reference
- CLIENT
- These functions are used by an mcp-client:
  - MarshalListToolsRequest
  - UnmarshalListToolsResult
  - MarshalCallToolRequest
  - UnmarshalCallToolResult

  - SERVER
  - all server functions should accept a "logger *utils.Logger" as a parameter. if its not there, add it. 
  - if the logger value is nil, return an error
- add these functions that aren't implemented yet
  - UnmarshalListToolsRequest
  - MarshalListToolsResult
  - UnmarshalCallToolRequest
  - MarshalCallToolResult