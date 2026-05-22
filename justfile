vet:
	go fmt ./...
	go vet ./...
	go test ./...

run *args:
	go run . -config dev-config.json {{args}}

docker-build:
	docker build -t simple-rss:local .
