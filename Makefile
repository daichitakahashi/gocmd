test:
	go test -coverprofile=coverage.out -p 1 -coverpkg=./... -v ./...

test-cov: test
	go tool cover -func=coverage.out
	rm coverage.out

test-cov-visual: test
	go tool cover -html=coverage.out
	rm coverage.out

.PHONY: test test-cov test-cov-visual
