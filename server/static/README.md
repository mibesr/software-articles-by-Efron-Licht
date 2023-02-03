# efron's blog source

## [cmd](./cmd)
command-line tools
## [articles](./articles)
raw markdown articles

## getting the source

```sh
git clone https://gitlab.com/efronlicht/blog
```

## running the server (docker)

```sh
# in the efronlicht/blog directory:
docker build -t efronlicht:latest
docker run -P 8080 efronlicht:latest
```

## running the server (local)
### requirements:
- go >= 1.19
- mermaid CLI
- docker