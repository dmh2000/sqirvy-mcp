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


update cmd/mcp-server/README.md file for the 'cmd/mcp-server' program. summarize its functionality and usage. do not add details of the code in the pkg directory. i will make a separate readme for that  

update pkg/mcp/README.md so the 'Functionality' section is organized as follows:
## Functionality
summary of functionality
### Client
#### Initialize
  describe each client side function in this section
#### Prompts
  describe each client side function in this section
#### Resources
  describe each client side function in this section
#### Tools
  describe each client side function in this section
### Server
#### Initialize
  describe each server side function in this section
#### Prompts
  describe each server side function in this section
#### Resources
  describe each server side function in this section
#### Tools
  describe each server side function in this section


in pkg/transport add a brief pkg/transport/README.md summarizing the functionality
in pkg/utils add a brief pkg/utils/README.md summarizing the functionality

update the top level README.md that summarizes this project. add a section with a brief description and a reference to cmd/mcp-server, pkg/mcp, pkg/transport and pkg/utils. 