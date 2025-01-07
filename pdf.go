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
	"regexp"
	"slices"
	"sort"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
	"github.com/gin-gonic/gin"
)

type PdfReturn struct {
	OutputFile  *os.File
	OutputFiles []string
}

type PdfStatus struct {
	success bool
	index   int
	result  *[]byte
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
