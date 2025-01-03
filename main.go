package main

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/gin-contrib/location"
	"github.com/gin-gonic/gin"
)

type PdfRequest struct {
	Data         []string  `json:"data,omitempty"`
	Url          []string  `json:"url,omitempty"`
	Download     bool      `json:"download"`
	Header       *string   `json:"header,omitempty"`
	Footer       *string   `json:"footer,omitempty"`
	MarginTop    *float32  `json:"marginTop,omitempty"`
	MarginBottom *float32  `json:"marginBottom,omitempty"`
	MarginLeft   *float32  `json:"marginLeft,omitempty"`
	MarginRight  *float32  `json:"marginRight,omitempty"`
	PaperSize    []float64 `json:"paperSize,omitempty"`
}

func getPdf(c *gin.Context) {
	var pdfRequestParams PdfRequest

	err := c.ShouldBindJSON(&pdfRequestParams)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"msg": "Could not bind", "message": err.Error()}) // This err.Error() gives away we're using golang and will need to be changed
		return
	}

	options, ok := c.MustGet("serverOptions").(*ServerOptions)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unable to retrieve data to generate PDF!", "message": "Error retrieving ServerOptions"})
		return
	}

	pdfResult, err := buildPdf(&pdfRequestParams, options)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Unable to generate PDF!", "message": err.Error()})
		return
	}

	if pdfRequestParams.Download {
		c.FileAttachment(pdfResult.OutputFile.Name(), "output.pdf")
		return
	}

	var outputFiles []string
	outFileName := filepath.Base(pdfResult.OutputFile.Name())
	serverUrl := location.Get(c)
	url := serverUrl.Scheme + "://" + serverUrl.Host + "/pdfs/"

	for _, value := range pdfResult.OutputFiles {
		outputFiles = append(outputFiles, url+filepath.Base(value))
	}

	c.IndentedJSON(http.StatusOK, gin.H{"url": url + outFileName, "pdf": outFileName, "components": outputFiles})
}

func getPdfPreview(c *gin.Context) {
	var pdfRequestParams PdfRequest
	err := c.ShouldBindJSON(&pdfRequestParams)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"msg": "Could not bind", "message": err.Error()}) // This err.Error() gives away we're using golang and will need to be changed
		return
	}

	options, ok := c.MustGet("serverOptions").(*ServerOptions)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unable to retrieve data to generate PDF!", "message": "Error retrieving ServerOptions"})
		return
	}

	pdfResult, err := buildPdf(&pdfRequestParams, options)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Unable to generate PDF!", "message": err.Error()})
		return
	}

	baseName, err := createPreviews(pdfResult.OutputFile.Name(), *options.RootDirectory+"/files/previews/")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Unable to generate PDF!", "message": err.Error()})
		return
	}

	pdfInfo, err := getPdfInfo(pdfResult.OutputFile.Name())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Unable to generate PDF!", "message": err.Error()})
		return
	}

	pages, err := strconv.ParseInt(pdfInfo["pages"], 10, 8)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Unable to generate PDF!", "message": "Unable to compute number of pages"})
		return
	}
	serverUrl := location.Get(c)
	url := serverUrl.Scheme + "://" + serverUrl.Host + "/preview/"

	var images []string
	for i := range pages {
		images = append(images, fmt.Sprintf("%s%s-%d.jpg", url, baseName, i+1))
	}

	c.IndentedJSON(http.StatusOK, gin.H{"success": true, "pages": pages, "images": images, "basename": baseName, "pdfInfo": pdfInfo})
}

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

	router := gin.Default()
	router.SetTrustedProxies(nil)
	router.Use(ApiMiddleware(serverOptions))
	router.Use(location.Default())
	router.POST("/pdf", getPdf)
	router.POST("/preview", getPdfPreview)
	router.GET("/status", getStatus)
	router.Static("/pdfs", *serverOptions.RootDirectory+"/files/pdfs")
	router.Static("/preview", *serverOptions.RootDirectory+"/files/previews")

	address := serverOptions.Address + fmt.Sprintf(":%d", serverOptions.Port)
	if serverOptions.UseTLS {
		router.RunTLS(address, *serverOptions.CertPath, *serverOptions.KeyPath)
		return
	}

	router.Run(address)
}
