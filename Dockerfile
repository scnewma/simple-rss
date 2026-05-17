FROM golang:1.25-bookworm AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build \
    -trimpath \
    -ldflags='-s -w' \
    -o /out/simple-rss .

FROM gcr.io/distroless/base-debian12:nonroot

COPY --from=build /out/simple-rss /simple-rss

EXPOSE 8080

ENTRYPOINT ["/simple-rss"]
