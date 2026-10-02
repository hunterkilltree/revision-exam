.PHONY: dev backend frontend test lint
backend:
	cd backend && go run ./cmd/server
frontend:
	cd frontend && npm run dev
dev:
	$(MAKE) -j2 backend frontend
test:
	cd backend && go test -race ./...
	cd frontend && npm test
lint:
	cd backend && go vet ./...
	cd frontend && npm run lint
