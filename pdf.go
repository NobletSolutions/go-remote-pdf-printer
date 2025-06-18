package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
	"github.com/gin-contrib/location"
	"github.com/gin-gonic/gin"
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

type PdfPreviewResponse struct {
	Pages   int8     `json:"pages"`
	Images  []string `json:"images"`
	pdfInfo map[string]string
}

type PdfReturn struct {
	OutputFile  *os.File
	OutputFiles []string
}

type PdfStatus struct {
	success bool
	index   int
	result  *[]byte
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

func getBrowserTargets(c *gin.Context) chromedp.Tasks {
	return chromedp.Tasks{
		chromedp.ActionFunc(func(ctx context.Context) error {
			infos, err := target.GetTargets().Do(ctx)
			if err != nil {
				return err
			}
			c.IndentedJSON(http.StatusOK, gin.H{"targets": infos})
			return nil
		}),
	}
}

func buildPdf(pdfRequestParams *PdfRequest, serverOptions *ServerOptions) (*PdfReturn, error) {
	requestData := pdfRequestParams.Data
	if serverOptions.DebugSources {
		tempFile, err := os.CreateTemp(*serverOptions.RootDirectory+"/files/sources/", "*.html")
		if err == nil {
			b, err := json.Marshal(requestData)
			if err == nil {
				os.WriteFile(tempFile.Name(), b, 0640)
			}
		}
	}

	printOptions, err := getPrintOptions(pdfRequestParams, &serverOptions.HeaderStyleTemplate)
	if err != nil {
		return nil, errors.New("parameter conversion error")
	}

	// build context options
	var opts []chromedp.ContextOption
	opts = append(opts, chromedp.WithLogf(log.Printf))
	opts = append(opts, chromedp.WithErrorf(log.Printf))

	if serverOptions.Debug {
		opts = append(opts, chromedp.WithDebugf(log.Printf))
	}

	channel := make(chan PdfStatus)
	for index, requestDataOrUrl := range requestData {
		allocatorContext, _ := chromedp.NewRemoteAllocator(context.Background(), "ws://"+serverOptions.ChromeUri)

		// create context
		ctx, cancel := chromedp.NewContext(allocatorContext, opts...)

		go chromedp.Run(ctx, printToPDF(requestDataOrUrl, printOptions, index, channel, cancel))
	}

	outputFiles := make(map[int]string)

	result := make([]PdfStatus, len(requestData))
	for i := range result {
		result[i] = <-channel
		if result[i].success {
			tempFile, err := os.CreateTemp(*serverOptions.RootDirectory+"/files/pdfs/", fmt.Sprintf("%d-*.pdf", i))
			if err != nil {
				return nil, errors.New("unable to create output file")
			}

			outputFiles[result[i].index] = tempFile.Name()
			os.WriteFile(tempFile.Name(), *result[i].result, 0640)
		}
	}

	keys := slices.Collect(maps.Keys(outputFiles))
	sort.Ints(keys)
	outputs := make([]string, len(keys))
	for _, k := range keys {
		outputs[k] = outputFiles[k]
	}

	// Merge the PDF files
	combinedFile, err := combinePdfs(outputs, serverOptions)
	if err != nil {
		return nil, errors.New("unable to combine component pdfs")
	}

	return &PdfReturn{OutputFile: combinedFile, OutputFiles: outputs}, nil
}

func printToPDF(urlStr string, params *page.PrintToPDFParams, index int, res chan PdfStatus, cancelContext context.CancelFunc) chromedp.Tasks {
	var base64EncodedData string
	match, _ := regexp.MatchString("(?i)^(https?|file|data):", urlStr)
	if match {
		base64EncodedData = urlStr
	} else {
		base64EncodedData = "data:text/html;base64," + base64.StdEncoding.EncodeToString([]byte(urlStr))
	}

	return chromedp.Tasks{
		chromedp.Navigate(base64EncodedData),
		chromedp.ActionFunc(func(ctx context.Context) error {
			buf, _, err := params.Do(ctx)
			if err != nil {
				res <- PdfStatus{false, index, nil}
			} else {
				res <- PdfStatus{true, index, &buf}
			}

			defer cancelContext()
			return nil
		}),
	}
}

func getPrintOptions(requestParams *PdfRequest, headerStyleTemplate *string) (*page.PrintToPDFParams, error) {
	params := page.PrintToPDF()
	params.PrintBackground = true

	// These are the default margins chrome has - but unless set uses no margins
	params.MarginTop = 0.4
	params.MarginBottom = 0.4
	params.MarginLeft = 0.39
	params.MarginRight = 0.39

	if requestParams.Header != nil {
		if requestParams.MarginTop == nil {
			return nil, errors.New("marginTop is required when providing a header template")
		}

		params.DisplayHeaderFooter = true
		params.HeaderTemplate = *headerStyleTemplate + *requestParams.Header
		params.FooterTemplate = "<footer></footer>"

		// accounts for the odd -0.16in margins
		var adjustment float64 = 0.35
		if *requestParams.MarginTop-1 > 0 {
			adjustment += 0.35 * (float64(*requestParams.MarginTop) - 1)
		}
		var top float64 = adjustment
		top += float64(*requestParams.MarginTop)
		params.MarginTop = top
	}

	if requestParams.Footer != nil {
		if requestParams.MarginBottom == nil {
			return nil, errors.New("marginBottom is required when providing a header template")
		}

		params.DisplayHeaderFooter = true
		params.FooterTemplate = *headerStyleTemplate + *requestParams.Footer

		if params.HeaderTemplate == "" {
			params.HeaderTemplate = "<header></header>"
		}

		// accounts for the odd -0.16in margins
		var adjustment float64 = 0.35
		if *requestParams.MarginBottom-1 > 0 {
			adjustment += 0.35 * (float64(*requestParams.MarginBottom) - 1)
		}

		var bottom float64 = adjustment
		bottom += float64(*requestParams.MarginBottom)
		params.MarginBottom = bottom
	}

	if requestParams.MarginLeft != nil {
		params.MarginLeft = float64(*requestParams.MarginLeft)
	}

	if requestParams.MarginRight != nil {
		params.MarginRight = float64(*requestParams.MarginRight)
	}

	if requestParams.MarginTop != nil {
		params.MarginTop = float64(*requestParams.MarginTop)
	}

	if requestParams.MarginBottom != nil {
		params.MarginBottom = float64(*requestParams.MarginBottom)
	}

	if len(requestParams.PaperSize) == 2 {
		params.PaperWidth = requestParams.PaperSize[0]
		params.PaperHeight = requestParams.PaperSize[1]
	}

	return params, nil
}

func getBrowserStatus(c *gin.Context, serverOptions *ServerOptions) {

	allocatorContext, allocatorCancel := chromedp.NewRemoteAllocator(context.Background(), "ws://"+serverOptions.ChromeUri)
	defer allocatorCancel()

	// build context options
	var opts []chromedp.ContextOption
	opts = append(opts, chromedp.WithLogf(log.Printf))

	// create context
	ctx, cancel := chromedp.NewContext(allocatorContext, opts...)
	defer cancel()

	// capture pdf
	if err := chromedp.Run(ctx, getBrowserTargets(c)); err != nil {
		// log.Fatal(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unable to retrieve status"})
	}
}
