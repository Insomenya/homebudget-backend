.PHONY: run build clean tidy migrate

APP  := homebudget
MAIN := ./cmd/server

run:
	go run $(MAIN)

build:
	go build -o $(APP) $(MAIN)

clean:
	rm -f $(APP) *.db

tidy:
	go mod tidy

migrate:
	go run ./cmd/migrate