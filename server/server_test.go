package main_test

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"testing"

	main "gitlab.com/efronlicht/blog/server"
)

func TestMain(m *testing.M) {
	os.Setenv("PORT", "6483")
	go main.Run()
	os.Exit(m.Run())
}

func get(p string) *http.Response {
	target := path.Join("http://localhost:6483/", p)
	resp, err := http.Get(target)
	if err != nil {
		panic(fmt.Errorf("get %s: %v", target, err))
	}
	return resp
}
