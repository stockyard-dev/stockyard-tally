build:
	CGO_ENABLED=0 go build -o tally ./cmd/tally/

run: build
	./tally

test:
	go test ./...

clean:
	rm -f tally

.PHONY: build run test clean
