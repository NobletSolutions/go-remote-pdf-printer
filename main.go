package main

import (
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"

	"github.com/NobletSolutions/go-remote-pdf-printer/docs"

	"github.com/gin-contrib/location"
	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type PdfRequest struct {
	Data         []string  `json:"data" form:"data"`
	Download     bool      `json:"download" form:"download"`
	Header       *string   `json:"header" form:"header"`
	Footer       *string   `json:"footer" form:"footer"`
	MarginTop    *float32  `json:"marginTop" form:"marginTop"`
	MarginBottom *float32  `json:"marginBottom" form:"marginBottom"`
	MarginLeft   *float32  `json:"marginLeft"  form:"marginLeft"`
	MarginRight  *float32  `json:"marginRight" form:"marginRight"`
	PaperSize    []float64 `json:"paperSize" form:"paperSize"`
}

type PdfResponse struct {
	Url        string   `json:"url"`
	Components []string `json:"components"`
}

type PreviewResponse struct {
	Pages   int8     `json:"pages"`
	Images  []string `json:"images"`
	pdfInfo map[string]string
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
		conf := c.PostFormMap("data")
		pdfRequestParams.Data = slices.Collect(maps.Values(conf))
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
// @Success 200 {object} PreviewResponse
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

	var images []string
	for i := range pages {
		images = append(images, fmt.Sprintf("%s%s-%d.jpg", url, baseName, i+1))
	}

	c.IndentedJSON(http.StatusOK, PreviewResponse{Pages: int8(pages), Images: images, pdfInfo: pdfInfo})
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
	router.GET("/status", getStatus)
	router.Static("/pdfs", *serverOptions.RootDirectory+"/files/pdfs")
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
