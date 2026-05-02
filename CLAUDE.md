# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What is lore

A self-hosted knowledge base powered by Claude Code sessions. Users upload `.jsonl` session logs, and Lore automatically generates documentation from them using an LLM. When related conversations are uploaded later, Lore uses LLM-based matching to find the existing document and updates it with new knowledge. Also works as a conversation browser with topic filtering.

Works as a standalone web app or from within Claude Code via MCP.

## Build and run

```bash
# Prerequisites: Go 1.24+, Node.js 20+, GCC (for SQLite CGo driver)

# Install frontend deps (first time or after package changes)
cd frontend && npm install && cd ..

# Build everything (frontend then Go binary)
make build

# Run the server (serves API + Vue SPA on :3000)
./lore serve

# Run with knowledge base features enabled
./lore serve --llm-provider anthropic --llm-api-key sk-ant-...

# Development: run backend and frontend separately for hot-reload
go run . serve                     # Terminal 1: Go API on :3000
cd frontend && npm run dev         # Terminal 2: Vite on :5173, proxies /api to :3000
```

There are no tests yet.

### CLI flags

| Flag | Env var | Default | Description |
|------|---------|---------|-------------|
| `--port` | — | `3000` | HTTP server port |
| `--db` | — | `lore.db` (next to binary) | SQLite database path |
| `--llm-provider` | `LORE_LLM_PROVIDER` | — | `anthropic` or `openai` |
| `--llm-api-key` | `LORE_LLM_API_KEY` | — | API key for the LLM provider |
| `--llm-model` | `LORE_LLM_MODEL` | Provider default | Model name override |
| `--llm-base-url` | `LORE_LLM_BASE_URL` | Provider default | API base URL override |

## Architecture

**Two entry points** (`main.go`): `lore serve` starts the HTTP server. `lore mcp` starts a JSON-RPC 2.0 stdio server for MCP integration with Claude Code, and also launches the HTTP server (and Vue SPA) in a background goroutine on the same `--port` so the web UI remains reachable while Claude Code is connected. If the port is already bound (e.g. `lore serve` is running elsewhere) the goroutine logs the bind error and the MCP stdio server keeps working.

### Conversation data flow

JSONL file → `parser.ParseFile` (line-by-line extraction, dedup, noise filtering) → `parser.MergeTurns` (collapse consecutive assistant messages, fold tool-result-only user turns) → `parser.DetectTopics` (30-minute gap heuristic between user messages) → `store.SaveConversation` (single transaction insert into SQLite).

### Knowledge base data flow

Upload conversation → `docgen.Pipeline.ProcessConversation`:

1. Segment conversation into topics using the LLM (`docgen.LLMSegmentTopics`), or fall back to time-gap topics if segmentation fails
2. Update DB topics with segments (`store.ReSegmentTopics`)
3. For each topic segment:
   a. Extract user/assistant text (`docgen.ExtractText`), skip if < 100 chars
   b. Ask the LLM to match the topic text against existing document titles
   c. **Match found** → pass existing doc + new topic text to LLM → update doc content
   d. **No match** → pass topic text to LLM → generate new doc

### Graceful degradation

| Configured | Behavior |
|-----------|----------|
| LLM configured | Full pipeline: segment topics, match to existing docs, generate/update docs |
| No LLM | Conversation viewer only — conversations saved but no docs generated |

### Backend packages (all under `internal/`)

- `db` — SQLite connection with WAL mode, inline schema creation (no migration files). Five tables: `conversations`, `topics`, `messages`, `documents`, `document_conversations`.
- `models` — Shared types. `ParsedConversation`/`ParsedTurn`/`ParsedTopic` (parser outputs), `Conversation`/`Topic`/`Message` (DB-mapped), `Document`/`DocumentConversation` (knowledge base types), `TopicSegment` (segmentation output).
- `parser` — JSONL parsing with regex-based noise filtering, turn merging, topic detection. `ParseConversation()` is the single entry point.
- `store` — CRUD on SQLite. `SaveConversation` uses upsert on `session_id`. `ReSegmentTopics` replaces time-gap topics with LLM segments. `documents.go` handles document CRUD, keyword search via `SearchDocuments`, and the `document_conversations` junction table.
- `server` — chi router with REST endpoints under `/api/`. Serves the Vue SPA from `frontend/dist` with SPA fallback. Upload handler triggers `docgen.Pipeline.ProcessConversation` (non-fatal on error).
- `mcp` — MCP stdio server exposing conversation tools and document tools. Upload tool triggers pipeline. Has its own duplicate `findSessionFile` function (same as `server/finder.go`). The HTTP server is started by `runMCP` in `main.go`, not from this package.
- `llm` — `Provider` interface with Anthropic (`api.anthropic.com/v1/messages`) and OpenAI (`api.openai.com/v1/chat/completions`) implementations. Direct HTTP, no SDK dependencies. `NewProvider()` returns nil if not configured.
- `docgen` — `Pipeline` orchestrates the segment→match→generate/update flow. `segment.go` does LLM-based topic segmentation with time-gap fallback. `text.go` extracts clean text from messages. `prompts.go` has system prompts for segmentation, matching, generate, and update operations. `pipeline.go` has `ProcessConversation` (triggered on upload) and `RegenerateDocument` (triggered via API).

**Frontend** (`frontend/`): Vue 3 + TypeScript + Vite. No component library — all custom components. Uses `marked` for markdown rendering and `highlight.js` for code block syntax highlighting. State management via Vue composables (`useDocuments.ts`, `useConversations.ts`). Document-centric UI: home page shows a doc grid, sidebar lists documents with search/delete, `DocumentView` renders markdown content, `ChatOverlay` slides in from the right to show source conversations with topic filtering.

**Plugin** (`plugin/`): Contains `.claude-plugin/plugin.json` (plugin metadata) and `.mcp.json` (MCP server config) for distributing Lore as a Claude Code plugin.

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

## MCP tools

| Tool | Description |
|------|-------------|
| `upload_conversation` | Upload by `session_id` or `path` (triggers doc generation) |
| `list_conversations` | List all conversations |
| `delete_conversation` | Delete conversation by ID |
| `list_documents` | List all generated documents |
| `get_document` | Get full document content by ID |
| `search_documents` | Keyword search over documents |
| `delete_document` | Delete document by ID |

## Known duplication

`findSessionFile` is implemented twice: `internal/server/finder.go` and `internal/mcp/tools.go`. Both walk `~/.claude/projects/` and fall back to `~/.claude/sessions/`.

## Testing Lore

### 1. Basic build and startup (no LLM)

```bash
make build
./lore serve
# Should show: LLM: disabled
# Conversation upload/browse works, no docs generated
```

### 2. Test full pipeline (with LLM)

```bash
# With Anthropic
./lore serve --llm-provider anthropic --llm-api-key sk-ant-...

# Or with environment variables
export LORE_LLM_PROVIDER=anthropic
export LORE_LLM_API_KEY=sk-ant-...
./lore serve

# Should show: LLM: enabled
```

Then test the document lifecycle:

```bash
# Upload a conversation — should generate a document
curl -X POST http://localhost:3000/api/conversations/upload \
  -H 'Content-Type: application/json' \
  -d '{"path": "/path/to/session.jsonl"}'

# Check that a document was created
curl http://localhost:3000/api/documents

# Read the generated document
curl http://localhost:3000/api/documents/1

# Upload a related conversation — should update the existing document
curl -X POST http://localhost:3000/api/conversations/upload \
  -H 'Content-Type: application/json' \
  -d '{"path": "/path/to/related-session.jsonl"}'

# Verify the document was updated (not duplicated)
curl http://localhost:3000/api/documents
# Should still show 1 document, now with 2 conversation IDs

# Upload an unrelated conversation — should create a second document
curl -X POST http://localhost:3000/api/conversations/upload \
  -H 'Content-Type: application/json' \
  -d '{"path": "/path/to/unrelated-session.jsonl"}'

# Should now show 2 documents
curl http://localhost:3000/api/documents

# Keyword search
curl -X POST http://localhost:3000/api/documents/search \
  -H 'Content-Type: application/json' \
  -d '{"query": "your search terms here", "limit": 5}'

# Regenerate a document from scratch
curl -X POST http://localhost:3000/api/documents/1/regenerate

# Delete a document
curl -X DELETE http://localhost:3000/api/documents/1

# Delete a conversation — document persists if other conversations link to it
curl -X DELETE http://localhost:3000/api/conversations/1
```

### 3. Verify database schema

```bash
sqlite3 lore.db ".tables"
# Should show: conversations  document_conversations  documents  messages  topics

sqlite3 lore.db "SELECT id, title FROM documents"
```
