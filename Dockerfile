# An annotated Dockerfile for `eblog` (https://eblog.fly.dev)
# Annotated July 2023 by Efron Licht
# See blog post at <TODO: add link> for more details.

# ---- PREFACE ---- 
# Your average build process is glacially slow. Docker was a tool invented to HELP with this problem,
# but it often makes it worse. This annotated guide will give you a real-world example
# of a dockerfile built for speed. It's the dockerfile that powers the blog you read this on.

# Most build processes can be broken down into 3 stages:
#   - obtaining or building the tools we need to compile our application
#   - compiling our application
#   - running that compiled application.
# On an ordinary machine, pre-containers, we could think of these as 
#   - setting up our dev environment & installing our tools (done a few times per machine)
#   - compiling our application (done many, many times per developer machine)
#   - running our application (done many times per developer machine, and many more times by users)
# In the 'real world', we would never accept having to reinstall Visual Studio or XCode every time we wanted to build our app.
# And our users would feel frustrated if we forced them to install many GiB of developer tools just to run our app.
# But we silently accept this with containers because it's just "the way things are".
#  **It isn't and we shouldn't.**
# By taking a little time and breaking the build process into those stages WITHIN our Dockerfile, 
# we can end up with much smaller, faster builds.


# golang:1.20.5-alpine3.18 contains the golang compiler & tools at a svelte 99.77 MB - more than 3x smaller than the 314.84 MB
# for golang:1.20.5. That'll save our CI system terrabytes in downloads and disk space over the years.


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
RUN go mod download && go build -o rendermd -trimpath ./cmd/rendermd\
&& go build -o buildindex -trimpath ./cmd/buildindex\
&& go build -o prezip -trimpath ./cmd/prezip\
&& go test ./...

# strip debug symbols from our tools to make them smaller,
# then remove 'strip' and other binutils we don't need anymore to save space
RUN strip ./rendermd ./buildindex ./prezip && apk del -r binutils

# at this point, we have all the tools we need to build our app,
# but we don't have the app's source code or it's assets.

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
RUN ./rendermd ./articles ./server/static\
&&  ./buildindex ./server/static\
&&  ./prezip ./server/static > ./server/static/assets.zip
RUN go mod download
RUN go test ./server/...
RUN go build -o /app -trimpath ./server



FROM alpine:3.17 as app 
# final layer: we just get the latest CA-certificates for https requests, and copy in and run our app.

# expose port 8080 so the webserver can listen to it
EXPOSE 8080 
# add ca-certificates to allow https requests.
# again, we use --no-cache, since this is the one and only time this image will access apk.
RUN apk --no-cache add ca-certificates
COPY --from=builder /app /app
ENTRYPOINT ./app