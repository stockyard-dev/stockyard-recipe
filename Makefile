build:
	CGO_ENABLED=0 go build -o recipe ./cmd/recipe/

run: build
	./recipe

test:
	go test ./...

clean:
	rm -f recipe

.PHONY: build run test clean
