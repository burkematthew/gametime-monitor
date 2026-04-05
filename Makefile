.PHONY: build build-pi build-pi64 serve run migrate-up migrate-down migrate-version clean

build:
	go build -o gametime-monitor ./cmd/gametime-monitor

build-pi:
	GOOS=linux GOARCH=arm GOARM=7 go build -o gametime-monitor-arm ./cmd/gametime-monitor

build-pi64:
	GOOS=linux GOARCH=arm64 go build -o gametime-monitor-arm64 ./cmd/gametime-monitor

serve:
	go run ./cmd/gametime-monitor serve

run:
	go run ./cmd/gametime-monitor run

migrate-up:
	go run ./cmd/gametime-monitor migrate up

migrate-down:
	go run ./cmd/gametime-monitor migrate down

migrate-version:
	go run ./cmd/gametime-monitor migrate version

clean:
	rm -f gametime-monitor gametime-monitor-arm gametime-monitor-arm64
