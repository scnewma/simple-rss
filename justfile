vet:
	go fmt ./...
	go vet ./...
	go test ./...

run:
	go run . -config dev-config.json

docker-build:
	docker build -t simple-rss:local .
