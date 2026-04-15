package main

import (
	"log"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

func ClearFileMiddleware(rootDirectory *string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Allow the static file to be served
		c.Next()

		// --- DO SOMETHING AFTER REQUEST ---
		clearValue := c.Query("clear")
		if clearValue != "" {

			u := c.Request.URL
			u.RawQuery = ""

			if strings.HasPrefix(u.Path, "/preview/") || strings.HasPrefix(u.Path, "/png/") || strings.HasPrefix(u.Path, "/pdfs/") {
				filePath := *rootDirectory + "/files" + u.Path
				log.Println(filePath)
				os.Remove(filePath)

				return
			}
		}
	}
}
