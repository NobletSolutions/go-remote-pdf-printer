package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/NobletSolutions/go-remote-pdf-printer/docs"

	"github.com/gin-contrib/location"
	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type PngResponse struct {
	Png string `json:"png"`
	Url string `json:"url"`
}

func extractData(c *gin.Context) (*PdfRequest, bool) {
	var pdfRequestParams PdfRequest

	// Handle JSON/XML/Form-Data
	err := c.ShouldBind(&pdfRequestParams)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Unable to extract request data", "details": err.Error()})
		return nil, false
	}

	if pdfRequestParams.Data == nil {
		formData := c.PostFormMap("data")

		// Getting the form is not guaranteed to come in submission order
		// We sort them, but need to sort numerically instead of via strings
		// otherwise we get sorted results like 0,1,10,11,...2,20.
		keys := make([]int, 0, len(formData))

		for k := range formData {
			i, err := strconv.Atoi(k)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Unable to extract request data", "details": err.Error()})
				return nil, false
			}
			keys = append(keys, i)
		}

		sort.Ints(keys)

		for _, key := range keys {
			a := strconv.Itoa(key)
			pdfRequestParams.Data = append(pdfRequestParams.Data, formData[a])
		}
	}

	if len(pdfRequestParams.Data) <= 0 {
		return nil, false
	}

	return &pdfRequestParams, true
}

// @Summary Submit urls/data to be converted to a PDF
// @Schemes
// @Description Submit urls/data to be converted to a PDF
// @Accept json
// @Accept xml
// @Produce json
// @Param data body PdfRequest true "The input todo struct"
// @Success 200 {object} PdfResponse
// @Failure      400
// @Failure      500
// @Router /pdf [post]
func getPdf(c *gin.Context) {
	options, ok := c.MustGet("serverOptions").(*ServerOptions)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unable to retrieve data to generate PDF!", "message": "Error retrieving ServerOptions"})
		return
	}

	pdfRequestParams, ok := extractData(c)
	if !ok {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"success": false, "error": "Unable to extract request data"})
		return
	}

	pdfResult, err := buildPdf(pdfRequestParams, options)
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

	c.IndentedJSON(http.StatusOK, PdfResponse{Url: url + outFileName, Components: outputFiles})
}

// @Summary Submit urls/data to be converted to a PDF and then one image per page
// @Schemes
// @Description Submit urls/data to be converted to a PDF and then one image per page
// @Accept json
// @Accept xml
// @Produce json
// @Param data body PdfRequest true "The input todo struct"
// @Success 200 {object} PdfPreviewResponse
// @Failure      400
// @Failure      500
// @Router /preview [post]
func getPdfPreview(c *gin.Context) {
	options, ok := c.MustGet("serverOptions").(*ServerOptions)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unable to retrieve data to generate PDF!", "message": "Error retrieving ServerOptions"})
		return
	}

	pdfRequestParams, ok := extractData(c)
	if !ok {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"success": false, "error": "Unable to extract request data"})
		return
	}

	pdfResult, err := buildPdf(pdfRequestParams, options)
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

	// pdftocairo prepends 0 to the page name, so we need to return the correct name prepended as well
	numberOfDigits := strconv.Itoa(int(pages))
	format := fmt.Sprintf("%s%s-%s%d%s.jpg", "%s", "%s", "%0", len(numberOfDigits), "d")

	var images []string
	for i := range pages {
		images = append(images, fmt.Sprintf(format, url, baseName, i+1))
	}

	c.IndentedJSON(http.StatusOK, PdfPreviewResponse{Pages: int8(pages), Images: images, pdfInfo: pdfInfo})
}

// @Summary Submit a single url or data to be converted to a png
// @Schemes
// @Description Submit a single url or data to be converted to a png
// @Accept json
// @Accept xml
// @Produce json
// @Param data body PngRequest true "The input request"
// @Success 200 {object} PngResponse
// @Failure      400
// @Failure      500
// @Router /png [post]
func getPng(c *gin.Context) {
	options, ok := c.MustGet("serverOptions").(*ServerOptions)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unable to retrieve data to generate screenshot!", "message": "Error retrieving ServerOptions"})
		return
	}

	var pngRequestParams PngRequest

	// Handle JSON/XML/Form-Data
	err := c.ShouldBind(&pngRequestParams)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Unable to extract request data", "details": err.Error()})
		return
	}

	if len(pngRequestParams.Data) <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "No Data", "details": "pngRequestParams.Data is empty"})
		return
	}

	outputFile, err := buildPng(&pngRequestParams, options)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Unable to generate screenshot!", "message": err.Error()})
		return
	}

	if pngRequestParams.Download {
		c.FileAttachment(outputFile.Name(), "output.pdf")
		return
	}

	outFileName := filepath.Base(outputFile.Name())
	serverUrl := location.Get(c)
	url := serverUrl.Scheme + "://" + serverUrl.Host + "/png/"

	c.IndentedJSON(http.StatusOK, PngResponse{Png: outFileName, Url: url + outFileName})
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
