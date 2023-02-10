
FROM golang:1.19.5-alpine3.17 as builder
EXPOSE 8080
ENV GOPATH=""
ENV CGO_ENABLED=0
# install mermaid CLI, needed by rendermd
RUN apk update && apk --no-cache add npm && npm install -g @mermaid-js/mermaid-cli
COPY . .
RUN go run ./cmd/rendermd . ./server/static/
RUN go run ./cmd/buildindex ./server/static
RUN go run ./cmd/prezip ./server/static
RUN go build -trimpath -o /app ./server 


FROM alpine:3.17
RUN apk --no-cache add ca-certificates
COPY --from=builder /app /app
ENTRYPOINT ./app