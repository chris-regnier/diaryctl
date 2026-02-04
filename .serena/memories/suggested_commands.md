# Suggested Commands

## Build & Run
```bash
go build -o diaryctl .
go run . <command>
```

## Test
```bash
go test ./...
go test ./cmd/ -v
go test ./internal/... -v
```

## Format & Lint
```bash
gofmt -w .
go vet ./...
```

## Run the app
```bash
./diaryctl daily          # Launch TUI picker
./diaryctl create "text"  # Create entry
./diaryctl jot "note"     # Quick note
./diaryctl today          # View today
./diaryctl list           # List entries
./diaryctl seed           # Generate test data
```
