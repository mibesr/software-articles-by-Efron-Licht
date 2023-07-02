SHELL=/bin/bash -o pipefail

all: build

deps:
	# make deps
	which flyctl || ./deps/install-fly.sh
generate: 
	# --- make generate ---
	git rev-parse HEAD > server/commit.txt # add current commit to server logs
	go run ./cmd/rendermd . ./server/static # generate static html from markdown
	go run ./cmd/buildindex ./server/static # build /index.html
	go run ./cmd/prezip ./server/static > ./server/static/assets.zip # zip up all of the assets

deps:  generate
	# --- make deps ----
	go mod tidy
	go mod download
	go mod vendor

test-css:
	# --- make test-css ---
	go run ./cmd/prezip ./server/static > ./server/static/assets.zip
	go run ./server

test: deps
	# --- make test ---
	go test ./...
	go build ./... # check that everything can build w/out compiler errors

build: test 
	# --- make build ---

	mkdir -p bin
	go build -o ./bin/efronblogserver ./server
run: test
	# --- make run ---
	go run ./server


deploy-test: deps 
 	# --- make deploy-test ---
	fly apps destroy -y eblog-test # remove the old app
	# deploy the new one
	fly launch \
	--auto-confirm \
	--copy-config \
	--name eblog-test \
	--local-only \
	--now \
	--region lax \
	--strategy immediate \
	--verbose

deploy: deps 
	go mod vendor
	fly apps destroy -y eblog # remove the old app
	# deploy the new one
	fly launch \
	--auto-confirm \
	--copy-config \
	--name eblog \
	--local-only \
	--now \
	--region lax \
	--strategy immediate \
	--verbose


