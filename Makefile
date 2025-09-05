.PHONY: test build run release lint wasm serve

test:
	go test -v ./...

build:
	go build -o chinchon .

wasm:
	GOOS=js GOARCH=wasm go build -tags tinygo -o main.wasm main_wasm.go

run:
	./chinchon

serve: wasm
	@echo "Starting web server at http://localhost:8080"
	@echo "Open your browser and go to http://localhost:8080 to play Chinchón!"
	python -m http.server 8080 2>/dev/null || python3 -m http.server 8080 2>/dev/null || go run -c "package main; import \"net/http\"; func main() { http.ListenAndServe(\":8080\", http.FileServer(http.Dir(\".\"))) }"

release:
	rm -rf dist && goreleaser

lint:
	golangci-lint run