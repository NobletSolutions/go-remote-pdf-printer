package main

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/NobletSolutions/go-remote-pdf-printer/docs"

	"github.com/gin-contrib/location"
	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func getStatus(c *gin.Context) {
	serverOptions, ok := c.MustGet("serverOptions").(*ServerOptions)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unable to retrieve data to generate PDF!", "message": "Error retrieving ServerOptions"})
		return
	}

	getBrowserStatus(c, serverOptions)
}

func main() {
	var serverOptions *ServerOptions
	serverOptions = New(serverOptions)
	createDirectories(serverOptions)

	gin.DisableConsoleColor()

	f, err := os.Create(serverOptions.LogPath + "/remote-pdf-printer.log")
	if err != nil {
		panic(fmt.Sprintln("Unable to open log file: " + err.Error()))
	}

	serverOptions.LogFile = f
	gin.DefaultWriter = io.MultiWriter(serverOptions.LogFile, os.Stdout)
	router := gin.Default()

	router.SetTrustedProxies(nil)
	router.Use(ApiMiddleware(serverOptions))
	router.Use(location.Default())
	if serverOptions.Debug {
		router.Use(LogRequestDataMiddleware(serverOptions))
	}

	router.POST("/pdf", getPdf)
	router.POST("/preview", getPdfPreview)
	router.POST("/png", getPng)
	router.GET("/status", getStatus)
	router.Static("/pdfs", *serverOptions.RootDirectory+"/files/pdfs")
	router.Static("/png", *serverOptions.RootDirectory+"/files/pngs")
	router.Static("/preview", *serverOptions.RootDirectory+"/files/previews")

	docs.SwaggerInfo.BasePath = "/"
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	address := serverOptions.Address + fmt.Sprintf(":%d", serverOptions.Port)
	if serverOptions.UseTLS {
		router.RunTLS(address, *serverOptions.CertPath, *serverOptions.KeyPath)
		return
	}

	router.Run(address)
}
