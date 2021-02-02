.PHONY: test
test:
	@go test -v .

gofmt:
	@gofmt -w *.go

