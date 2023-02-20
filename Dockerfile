
FROM golang:1.19.5-alpine3.17 as builder
EXPOSE 8080
ENV GOPATH=""
ENV CGO_ENABLED=0
COPY . .
RUN go run ./cmd/rendermd . ./server/static/
RUN go run ./cmd/buildindex ./server/static
RUN go run ./cmd/prezip ./server/static > ./server/static/assets.zip
RUN go test ./...
RUN go build -trimpath -o /app ./server 


FROM alpine:3.17
RUN apk --no-cache add ca-certificates
COPY --from=builder /app /app
ENTRYPOINT ./app