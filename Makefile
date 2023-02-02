SHELL=/bin/bash -o pipefail

all: build

deps:
	# make deps
	which flyctl || ./deps/install-fly.sh
generate: # make generate
	# --- make generate ---
	git rev-parse HEAD > server/commit.txt
	go run ./cmd/rendermd . ./server/static # generate static html from markdown

deps: generate 
	# --- make deps ----
	go mod tidy
	go mod download
	go mod vendor


test:
	# --- make test ---
	go test -v ./...
	go build ./... # check that everything can build w/out compiler errors

build: test 
	# --- make build ---

	mkdir -p bin
	go build -o ./bin/efronblogserver ./server
run: test # make run
	go run ./server

deploy: deps test # make deploy
	fly apps destroy -y eblog
	fly launch \
	--auto-confirm \
	--copy-config \
	--name eblog \
	--now \
	--region lax \
	--strategy immediate
	--verbose


