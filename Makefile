build:
	export GOBIN=$(shell pwd)
	go build

profile: build
	go test -run none -bench . -benchtime 4s -cpuprofile=prof.out
	go tool pprof ./automathub ./prof.out

run:
	go run main.go config.go tcp.go handlers.go metrics.go ws.go --race

todo:
	@grep -n TODO *.go || true

cover:
	go test -coverprofile=coverage.out
	go tool cover -html=coverage.out

clean:
	go clean
	rm *.out