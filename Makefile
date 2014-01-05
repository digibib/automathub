build:
	export GOBIN=$(shell pwd)
	go build

profile: build
	go test -run none -bench . -benchtime 4s -cpuprofile=prof.out
	go tool pprof ./automathub ./prof.out

run:
	go run main.go automat.go config.go tcp.go handlers.go metrics.go ws.go --race

todo:
	@grep -rn TODO * || true
	@grep -rn println * || true

test:
	go test

integration:
	go test -tags integration

cover:
	go test -tags integration -coverprofile=coverage.out
	go tool cover -html=coverage.out

clean:
	go clean
	rm *.out