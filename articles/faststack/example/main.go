package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.GET("/panic", func(c *gin.Context) {
		fmt.Fprintf(c.Writer, "%s", f())
	})
	http.ListenAndServe(":8080", engine)
}

func f() string {
	panic("this function panics!")
}
