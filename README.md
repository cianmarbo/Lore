<p align="center">
  <img src="docs/lore-icon.png" alt="Lore" width="160">
</p>

# Lore

Lore is a living knowledge base powered by Claude Code sessions. 

You can upload your `.jsonl` session logs and Lore automatically generates documentation from them using the configured LLM. When you upload related conversations later, Lore uses LLM-based matching to find the existing document and updates it with new knowledge, or creates new documentation. 

You can upload conversations directly inside Claude Code using the stdio MCP server. Just ask Claude "could you upload this?", for example.

Lore also works as a conversation browser — view sessions in a chat UI with syntax-highlighted code blocks and filter by auto-detected topics.

Works as a standalone web app or directly from within Claude Code via MCP.

## Status

> [!WARNING]
> Lore is very early in development. There are no tests yet, the schema and APIs may change without notice, and rough edges are expected. Be vigilant — back up your `lore.db` before upgrading, and treat generated documents as a starting point rather than a source of truth.

## Prerequisites

- Go 1.24+
- Node.js 20+
- GCC (required by the SQLite CGo driver)

For the knowledge base features:
- An LLM API key (Anthropic or OpenAI)

## Build

```bash
# Install frontend dependencies and build everything
cd frontend && npm install && cd ..
make build
```

This produces a single `lore` binary that serves both the API and the Vue frontend.

## Run

### Conversation viewer only

```bash
./lore serve
```

Open [http://localhost:3000](http://localhost:3000). Upload and browse conversations — no LLM required.

### With knowledge base

```bash
# With Anthropic
./lore serve --llm-provider anthropic --llm-api-key sk-ant-...

# With OpenAI
./lore serve --llm-provider openai --llm-api-key sk-...

# Or use environment variables
export LORE_LLM_PROVIDER=anthropic
export LORE_LLM_API_KEY=sk-ant-...
./lore serve
```

When an LLM is configured, uploading a conversation automatically generates a document. Lore uses the LLM to segment conversations into topics and match them against existing documents, updating them with new knowledge rather than creating duplicates.

### Options

| Flag | Env var | Default | Description |
|------|---------|---------|-------------|
| `--port` | — | `3000` | HTTP server port |
| `--db` | — | `lore.db` (next to binary) | SQLite database path |
| `--llm-provider` | `LORE_LLM_PROVIDER` | — | `anthropic` or `openai` |
| `--llm-api-key` | `LORE_LLM_API_KEY` | — | API key for the LLM provider |
| `--llm-model` | `LORE_LLM_MODEL` | Provider default | Model name override |
| `--llm-base-url` | `LORE_LLM_BASE_URL` | Provider default | API base URL override |

On startup, Lore prints the status of each subsystem:

```
lore server starting
  Database: lore.db
  Address:  http://localhost:3000
  Frontend: serving from frontend/dist
  LLM: enabled
```

## Upload a conversation

### Via the API

```bash
# By file path
curl -X POST http://localhost:3000/api/conversations/upload \
  -H 'Content-Type: application/json' \
  -d '{"path": "/path/to/session.jsonl"}'

# By session ID (auto-searches ~/.claude/ for the file)
curl -X POST http://localhost:3000/api/conversations/upload \
  -H 'Content-Type: application/json' \
  -d '{"session_id": "a77a49ed-5860-46fe-a842-be3b161ae6a1"}'
```

If an LLM is configured, the upload response will also trigger document generation in the background.

### From within Claude Code (MCP)

Register the MCP server:

```bash
claude mcp add --transport stdio lore -- /full/path/to/lore mcp
```

If using the knowledge base features, pass the flags:

```bash
claude mcp add --transport stdio lore -- /full/path/to/lore mcp \
  --llm-provider anthropic --llm-api-key sk-ant-...
```

`lore mcp` also starts the web UI in the background on `--port` (default `3000`), so you can browse uploaded conversations and generated documents at [http://localhost:3000](http://localhost:3000) while Claude Code is connected. Pass `--port` to change it. If the port is already in use (e.g. `lore serve` is running elsewhere) the MCP server still works — only the web UI is unavailable.

Then in any Claude Code session you can say things like:

- "Upload this conversation"
- "What conversations have I uploaded?"
- "Search my knowledge base for Go concurrency patterns"
- "Show me document 3"
- "Delete conversation 2"

Claude discovers the tools automatically and maps your natural language to the right MCP tool call.

#### MCP tools

| Tool | Description |
|------|-------------|
| `upload_conversation` | Upload a `.jsonl` session by path or session ID |
| `list_conversations` | List all uploaded conversations |
| `delete_conversation` | Delete a conversation by ID |
| `list_documents` | List all generated knowledge base documents |
| `get_document` | Get full document content by ID |
| `search_documents` | Keyword search over documents by query |
| `delete_document` | Delete a document by ID |

## How it works

### Conversation parsing

1. Claude Code sessions are stored as `.jsonl` files under `~/.claude/projects/`
2. The parser reads these line-by-line, extracts user/assistant/system messages, deduplicates streaming chunks, merges consecutive assistant turns, and folds tool-result-only user messages into the preceding assistant turn. Topics are first set from 30-minute gaps between user messages; if an LLM is configured, the document pipeline replaces those with LLM-derived segments before they're surfaced in the UI
3. Parsed conversations are stored in SQLite with full-text topic labels for search
4. The Vue frontend fetches data via the REST API and renders it as a chat UI with syntax highlighting (highlight.js) and collapsible tool call details

### Knowledge base generation

When an LLM is configured, each uploaded conversation triggers a document generation pipeline:

1. The conversation is segmented into topics by the LLM (falls back to 30-minute time gaps if LLM segmentation fails)
2. Each topic segment is processed independently:
   a. User/assistant message text is extracted, filtering out noise (acknowledgments, tool orchestration, slash commands)
   b. The LLM is asked to match the topic against existing document titles
   c. **Match found** — the existing document and the new topic are sent to the LLM, which updates the document with new knowledge
   d. **No match** — the topic is sent to the LLM, which generates a new document

This means your knowledge base grows and refines itself as you upload more sessions. Related topics — even across different conversations — merge into a single evolving document rather than creating duplicates.

### Graceful degradation

| Configuration | Behavior |
|--------------|----------|
| LLM configured | Full pipeline: segment topics, match to existing docs, generate/update documents |
| No LLM | Conversation viewer only — conversations saved but no documents generated |

## REST API

### Conversations

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/conversations` | List all conversations |
| POST | `/api/conversations/upload` | Upload by `path` or `session_id` |
| GET | `/api/conversations/{id}` | Get conversation with topics |
| DELETE | `/api/conversations/{id}` | Delete conversation (cascades) |
| GET | `/api/conversations/{id}/messages` | Get messages, optional `?topic_id=` filter |
| GET | `/api/search?q=` | Search topics by label (LIKE match) |

### Documents

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/documents` | List all documents (title, IDs, no content) |
| GET | `/api/documents/{id}` | Get document with full content + linked conversation IDs |
| DELETE | `/api/documents/{id}` | Delete document |
| POST | `/api/documents/search` | Keyword search: `{"query": "...", "limit": 5}` |
| POST | `/api/documents/{id}/regenerate` | Re-generate document from all linked conversations |

## Development

Run the Go backend and Vite dev server separately for hot-reloading:

```bash
# Terminal 1: Go API server
go run . serve

# Terminal 2: Vite dev server (proxies /api to Go)
cd frontend && npm run dev
```

The Vite dev server runs on `:5173` and proxies API calls to the Go server on `:3000`.

## Project structure

```
lore/
  main.go                    CLI entry point (serve / mcp)
  internal/
    db/                      SQLite connection + schema
    models/                  Shared data types
    parser/                  JSONL parsing, turn merging, topic detection
    store/                   CRUD operations (conversations + documents + keyword search)
    server/                  HTTP router + REST API handlers
    mcp/                     MCP stdio server (JSON-RPC); web UI is started alongside it from main.go
    llm/                     LLM provider interface (Anthropic + OpenAI via HTTP)
    docgen/                  Document generation pipeline (segment, match, generate, update)
  frontend/
    src/
      components/            Vue components (Sidebar, DocumentView, ChatOverlay, ChatView, etc.)
      composables/           Reactive state management (useDocuments, useConversations)
      api.ts                 API client
      types.ts               TypeScript interfaces
  plugin/                    Claude Code plugin metadata + MCP config
```
