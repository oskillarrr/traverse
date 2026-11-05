package main

import (
	"example.com/gosvc/internal/handlers"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	v1 := r.Group("/api/v1")
	v1.GET("/articles/:slug", handlers.Get)
}
