---
title: Claude Code Setup
sidebar:
  order: 2
---

Configure gopls-mcp with Claude Code.

---

## Step 1: Build or Install gopls-mcp

Ensure [claude code]() is installed, and run command below to add mcp server into claude code.

```bash
claude mcp add gopls-mcp -- gopls-mcp
```

You will see the similar log below if succeed.

```
Added stdio MCP server gopls-mcp with command: gopls-mcp to local config
File modified: /home/xieyuschen/.claude.json
[project: /home/xieyuschen/codespace/gopls-mcproot]
```

---

## Step 2: Verify Installation

Inside claude code, run `/mcp` command to verify `gopls-mcp` is availble.

```
(claude)> /mcp
```

If the tool is successfully added, you will see similiar output below:

```
 Manage MCP servers
 1 server

   Local MCPs (/home/xieyuschen/.claude.json
   [project: /home/xieyuschen/codespace/gopls-mcproot])
 ❯ gopls-mcp · ✔ connected

 https://code.claude.com/docs/en/mcp for help
```

---

## Next Steps

- [**Available Tools**](tools.md) - Learn about all 15 MCP tools
- [**Tool Reference**](../reference/index.md) - Detailed tool documentation
