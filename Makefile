all:
	go vet
	golint .
	make todo

build:
	export GOBIN=$(shell pwd)
	go build

package: build
	tar -cvzf automathub.tar.gz automathub config.json data/

profile: build
	go test -run none -bench . -benchtime 4s -cpuprofile=prof.out
	go tool pprof ./automathub ./prof.out

run:
	go run main.go automat.go config.go tcp.go handlers.go metrics.go ws.go protocols.go sip.go pool.go --race

todo:
	@grep -rn TODO * || true
	@grep -rn println * || true

test:
	go test -i
	go test

integration:
	go test -tags integration

cover:
	go test -tags integration -coverprofile=coverage.out
	go tool cover -html=coverage.out

clean:
	go clean
	rm *.out