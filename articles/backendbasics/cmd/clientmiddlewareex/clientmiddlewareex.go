package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"gitlab.com/efronlicht/blog/articles/backendbasics/cmd/clientmw"
)

func clientMiddleware() http.RoundTripper {
	var rt clientmw.RoundTripFunc // specify the type as a RoundTripFunc, not a http.RoundTripper, so that we don't have to repeatedly wrap it in RoundTripFunc(rt)
	const wait, tries = 10 * time.Millisecond, 3
	// first middleware applied will be the last one to run.
	rt = clientmw.RetryOn5xx(http.DefaultTransport, wait, tries) // retry on 5xx status codes
	rt = clientmw.Log(rt)                                        // log request duration and status code; uses trace from next middleware
	rt = clientmw.Trace(rt)                                      // add trace id to request header
	return rt
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("target url required")
	}
	target := os.Args[1]
	client := &http.Client{Transport: clientMiddleware(), Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(context.TODO(), "GET", target, nil)
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	io.Copy(os.Stdout, resp.Body)
}
