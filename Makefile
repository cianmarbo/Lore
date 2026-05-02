.PHONY: dev frontend build clean

# Development: run Go server + Vite dev server
dev:
	@echo "Start the Go server in one terminal:"
	@echo "  go run . serve"
	@echo ""
	@echo "Start the Vite dev server in another:"
	@echo "  cd frontend && npm run dev"

# Build the Vue frontend
frontend:
	cd frontend && npm run build

# Build everything: frontend then Go binary
build: frontend
	go build -o lore .

clean:
	rm -rf frontend/dist lore
