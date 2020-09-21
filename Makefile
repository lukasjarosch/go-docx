GOLANG_CI_LINT_VERSION="v1.31.0"

lint:
	@docker run --rm -v $(pwd):/app -w /app golangci/golangci-lint:${GOLANG_CI_LINT_VERSION} golangci-lint run -v


.PHONY: test
test:
	@go test -v .