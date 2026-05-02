package main

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"lore/internal/db"
	"lore/internal/docgen"
	"lore/internal/llm"
	"lore/internal/mcp"
	"lore/internal/server"
	"lore/internal/store"
)

func defaultDBPath() string {
	exe, err := os.Executable()
	if err != nil {
		return "lore.db"
	}
	return filepath.Join(filepath.Dir(exe), "lore.db")
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: lore <serve|mcp> [options]\n\n")
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  serve [--port PORT] [--db PATH]   Start the web server\n")
		fmt.Fprintf(os.Stderr, "  mcp [--port PORT] [--db PATH]    Start MCP stdio server + web UI\n")
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	dbPath := defaultDBPath()
	port := "3000"
	llmCfg := llm.Config{}

	// Simple flag parsing
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--db":
			if i+1 < len(args) {
				dbPath = args[i+1]
				i++
			}
		case "--port":
			if i+1 < len(args) {
				port = args[i+1]
				i++
			}
		case "--llm-provider":
			if i+1 < len(args) {
				llmCfg.Provider = args[i+1]
				i++
			}
		case "--llm-api-key":
			if i+1 < len(args) {
				llmCfg.APIKey = args[i+1]
				i++
			}
		case "--llm-model":
			if i+1 < len(args) {
				llmCfg.Model = args[i+1]
				i++
			}
		case "--llm-base-url":
			if i+1 < len(args) {
				llmCfg.BaseURL = args[i+1]
				i++
			}
		}
	}

	// Environment variable fallbacks for LLM config
	if llmCfg.Provider == "" {
		llmCfg.Provider = os.Getenv("LORE_LLM_PROVIDER")
	}
	if llmCfg.APIKey == "" {
		llmCfg.APIKey = os.Getenv("LORE_LLM_API_KEY")
	}
	if llmCfg.Model == "" {
		llmCfg.Model = os.Getenv("LORE_LLM_MODEL")
	}
	if llmCfg.BaseURL == "" {
		llmCfg.BaseURL = os.Getenv("LORE_LLM_BASE_URL")
	}

	database, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	st := store.New(database)

	// Create LLM provider (nil if not configured)
	llmProvider := llm.NewProvider(llmCfg)

	log.Printf("startup: llm=%v provider=%q", llmProvider != nil, llmCfg.Provider)

	// Document generation pipeline
	pipeline := docgen.New(st, llmProvider)

	switch command {
	case "serve":
		runServe(st, pipeline, llmProvider, port, dbPath)
	case "mcp":
		runMCP(st, pipeline, port)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		os.Exit(1)
	}
}

func runServe(st *store.Store, pipeline *docgen.Pipeline, llmProvider llm.Provider, port, dbPath string) {
	// Try to use embedded frontend, fall back to nil (API-only mode)
	var staticFS fs.FS
	distPath := "frontend/dist"
	if info, err := os.Stat(distPath); err == nil && info.IsDir() {
		staticFS = os.DirFS(distPath)
	}

	handler := server.New(st, staticFS, pipeline)

	fmt.Printf("lore server starting\n")
	fmt.Printf("  Database: %s\n", dbPath)
	fmt.Printf("  Address:  http://localhost:%s\n", port)
	if staticFS != nil {
		fmt.Printf("  Frontend: serving from %s\n", distPath)
	} else {
		fmt.Printf("  Frontend: not found (API-only mode)\n")
	}
	if llm.Available(llmProvider) {
		fmt.Printf("  LLM: enabled\n")
	} else {
		fmt.Printf("  LLM: disabled (set --llm-provider and --llm-api-key)\n")
	}
	fmt.Println()

	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func runMCP(st *store.Store, pipeline *docgen.Pipeline, port string) {
	// Start the HTTP server in the background so the web UI is available
	var staticFS fs.FS
	exeDir := filepath.Dir(os.Args[0])
	for _, candidate := range []string{
		filepath.Join(exeDir, "frontend", "dist"),
		"frontend/dist",
	} {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			staticFS = os.DirFS(candidate)
			break
		}
	}

	handler := server.New(st, staticFS, pipeline)
	go func() {
		log.Printf("mcp: web UI available at http://localhost:%s", port)
		if err := http.ListenAndServe(":"+port, handler); err != nil {
			log.Printf("mcp: http server error: %v", err)
		}
	}()

	srv := mcp.NewServer(st, pipeline)
	if err := srv.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "MCP error: %v\n", err)
		os.Exit(1)
	}
}
