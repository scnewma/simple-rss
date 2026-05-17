vet:
	go fmt ./...
	go vet ./...
	go test ./...

run:
	[ -f dev.db ] || go run ./tools/devdb -out dev.db
	go run . -config dev-config.json

dev-db:
	go run ./tools/devdb -out dev.db

docker-build:
	docker build -t simple-rss:local .
