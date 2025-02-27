package main

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

func buildPng(pngRequestParams *PngRequest, serverOptions *ServerOptions) error {
	requestData := pngRequestParams.Data
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

	var screenshotBuffer []byte
	err := chromedp.Run(ctx,
		chromedp.Navigate("https://www.scrapingcourse.com/ecommerce/product/adrienne-trek-jacket/"),
		chromedp.FullScreenshot(&screenshotBuffer, 100),
	)
	if err != nil {
		return err
	}

	// file permissions: 0644 (Owner: read/write, Group: read, Others: read)
	// write the response body to an image file
	err = os.WriteFile("full-page-screenshot.png", screenshotBuffer, 0644)
	if err != nil {
		return err
	}

	return nil
}

func getScreenshotOptions(requestParams *PdfRequest, headerStyleTemplate *string) (*page.PrintToPDFParams, error) {
	params := page.CaptureScreenshotFormatPng
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
