# see https://eblog.fly.dev/fastdocker.html for an article explaining this file.

FROM golang:1.20.5-alpine3.18 as tooling

# enable module mode & disable cgo
ENV GOPATH=""
ENV CGO_ENABLED=0

# install tools via package manager (apk, brew, apt, etc).
# we don't cache, since this will be the ONLY run of this step from the viewpoint of the container.
# binutils shouldn't ever change, so we put this first.
# we want it to strip our tools' binaries to make our image even smaller.

RUN apk add --no-cache binutils


# -- dependencies --
# any change to dependencies will invalidate the cache for this layer,
# but that's ok, because we want to update our dependencies anyway: they
# might also be shared by the tools.

# a note about copying:
# blanket copies of whole folders (or worse, whole projects) are a quick way to waste space & time.
# any change to a copied file invalidates the cache for this layer.
# if you do something like COPY . ., ANY change to ANY file in the project will invalidate the cache.
# this is slow and wasteful  only copy what you need.

# first, we check for any changes to dependencies.
COPY ./go.mod ./go.mod
COPY ./go.sum ./go.sum
# copy tools & libraries for tooling 
COPY ./observability ./observability
COPY ./cmd ./cmd

# -- build our tools--
# extra layers are very cheap. for steps that are all part of the same logical 'unit',
# you can combine them into a single step to save a few bytes, but 
# generally speaking, if you're not sure, just make a new COPY or RUN step.
RUN --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg/mod go mod download && go build -o rendermd -trimpath ./cmd/rendermd\
&& go build -o buildindex -trimpath ./cmd/buildindex\
&& go build -o prezip -trimpath ./cmd/prezip\
&& go test ./...

# strip debug symbols from our tools to make them smaller,
# then remove 'strip' and other binutils we don't need anymore to save space
RUN strip ./rendermd ./buildindex ./prezip\
&& apk del -r binutils
  
# at this point, we have all the tools we need to build our app,
# but we don't have the app's source code or it's assets.

# -- END OF TOOLING STAGE / BEGIN BUILD STAGE ---

FROM tooling as builder 
# builder uses the previously-built tools to build the app.
# since we embed the articles directly into the binary,
# any change to either means we need to rebuild the app.
# but since we're starting from an image with a warm cache and 
# all the dependencies and tools already installed, this is fast.

COPY ./server ./server
COPY ./articles ./articles
COPY .git/logs/refs/heads/master server/commit.txt
# run the tools we built during the tooling stage. see their source for details, but:
#   - rendermd renders the articles into html
#   - buildindex builds the homepage /index.html
#   - prezip zips up all the assets for storage & serving (since most of our clients have Accept-Encoding: deflate)
RUN --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg/mod ./rendermd ./articles ./server/static\
&&  ./buildindex ./server/static\
&&  ./prezip ./server/static > ./server/static/assets.zip
RUN --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg/mod go mod download\
&& go build -o /app -trimpath ./server

# --- END OF BUILD STAGE / BEGIN APP STAGE ---

FROM alpine:3.18.2 as app 
# final layer: we just get the latest CA-certificates for https requests, and copy in and run our app.

# expose port 8080 so the webserver can listen to it
EXPOSE 8080 
COPY --from=builder /app /app
# add ca-certificates to allow https requests.
# again, we use --no-cache, since this is the one and only time this image will access apk.
# RUN apk --no-cache add ca-certificates
ENTRYPOINT ./app
