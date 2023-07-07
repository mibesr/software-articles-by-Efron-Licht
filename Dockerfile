# layers: tooling, builder, app.
# tooling changes least often: it's the tools & libraries we use to build the app.
# builder changes when the articles or server change.
# app just contains the binary (w/ embedded statica assets) and latest certificates.

FROM golang:1.19.5-alpine3.17 as tooling
# build our tools in the first layer.
# these are unlikely to change often, so our cache is likely to remain good.

# expose port 8080 for the webserver
EXPOSE 8080

# -- dependencies --
# any change to dependencies will invalidate the cache for this layer,
# but that's ok, because we want to update our dependencies anyway: they
# might also be shared by the tools.

COPY ./go.mod ./go.mod
COPY ./go.sum ./go.sum
# copy tools & libraries for tooling 
COPY ./observability ./observability
COPY ./cmd ./cmd
ENV GOPATH=""
ENV CGO_ENABLED=0
# -- build our tools--
RUN go mod download && go build -o rendermd -trimpath ./cmd/rendermd\
&& go build -o buildindex -trimpath ./cmd/buildindex\
&& go build -o prezip -trimpath ./cmd/prezip\
&& go test ./...

FROM tooling as builder 
# builder uses the previously-built tools to build the app.
# since we embed the articles directly into the binary,
# any change to either means we need to rebuild the app.
# we've already built the tools & downloaded the dependencies, though,
# so this should be fast.
COPY ./articles ./articles
COPY ./server ./server
COPY .git/logs/refs/heads/master server/commit.txt
# run the tools to generate the static assets
RUN ./rendermd ./articles ./server/static\
&&  ./buildindex ./server/static\
&&  ./prezip ./server/static > ./server/static/assets.zip
RUN go mod download
RUN go test ./server/...
RUN CGO_ENABLED=0 go build -o /app -trimpath ./server


FROM alpine:3.17
ENV GOPATH=""
# add ca-certificates to allow https requests
RUN apk --no-cache add ca-certificates
COPY --from=builder /app /app
ENTRYPOINT ./app