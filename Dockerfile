
FROM golang:1.19.5-alpine3.17 as builder
# expose port 8080 for the webserver
EXPOSE 8080

ENV GOPATH=""
# disable cgo to create a static binary
ENV CGO_ENABLED=0
COPY . .
# render markdown files to http and highlight syntax
RUN go run ./cmd/rendermd . ./server/static/
# create ./server/static/index.html
RUN go run ./cmd/buildindex ./server/static 
# create ./server/static/assets.zip from all files in ./server/static
RUN go run ./cmd/prezip ./server/static > ./server/static/assets.zip

RUN go test ./...
# -trimpath removes the absolute path from the binary
# -o /app sets the output file to /app
RUN go build -trimpath -o /app ./server 


FROM alpine:3.17
# add ca-certificates to allow https requests
RUN apk --no-cache add ca-certificates
COPY --from=builder /app /app
ENTRYPOINT ./app