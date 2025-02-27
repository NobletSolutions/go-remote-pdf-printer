package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"os"
	"regexp"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

type PngRequest struct {
	Data     string   `json:"data" form:"data"`
	Download bool     `json:"download" form:"download"`
	X        *float32 `json:"x" form:"x"`
	Y        *float32 `json:"y" form:"y"`
	Width    *float32 `json:"width" form:"width"`
	Height   *float32 `json:"height" form:"height"`
	Scale    *float32 `json:"scale" form:"scale"`
}

func buildPng(pngRequestParams *PngRequest, serverOptions *ServerOptions) (*os.File, error) {
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

	var err error
	var printOptions *page.CaptureScreenshotParams

	printOptions, err = getScreenshotOptions(pngRequestParams)
	if err != nil {
		log.Printf("Error: %s", err.Error())
		return nil, err
	}

	var base64EncodedData string
	match, _ := regexp.MatchString("(?i)^(https?|file|data):", pngRequestParams.Data)
	if match {
		base64EncodedData = pngRequestParams.Data
	} else {
		base64EncodedData = "data:text/html;base64," + base64.StdEncoding.EncodeToString([]byte(pngRequestParams.Data))
	}

	// build context options
	var opts []chromedp.ContextOption
	opts = append(opts, chromedp.WithLogf(log.Printf))
	opts = append(opts, chromedp.WithErrorf(log.Printf))

	if serverOptions.Debug {
		opts = append(opts, chromedp.WithDebugf(log.Printf))
	}

	allocatorContext, _ := chromedp.NewRemoteAllocator(context.Background(), "ws://"+serverOptions.ChromeUri)

	// create context
	ctx, cancel := chromedp.NewContext(allocatorContext, opts...)
	defer cancel()

	var screenshotBuffer []byte
	err = chromedp.Run(ctx,
		chromedp.Navigate(base64EncodedData),
		printToPng(&screenshotBuffer, printOptions),
	)

	if err != nil {
		return nil, err
	}

	sz := len(screenshotBuffer)
	log.Printf("Screenshot Buffer Length %d", sz)

	if sz <= 0 {
		return nil, errors.New("no image returned")
	}

	tempFile, err := os.CreateTemp(*serverOptions.RootDirectory+"/files/pngs/", "*.png")
	if err != nil {
		return nil, errors.New("unable to create output file")
	}

	os.WriteFile(tempFile.Name(), screenshotBuffer, 0640)

	return tempFile, nil
}

func printToPng(res *[]byte, params *page.CaptureScreenshotParams) chromedp.Action {
	if res == nil {
		panic("res cannot be nil")
	}

	return chromedp.ActionFunc(func(ctx context.Context) error {
		buf, err := params.Do(ctx)

		*res = buf

		return err
	})
}

func getScreenshotOptions(requestParams *PngRequest) (*page.CaptureScreenshotParams, error) {
	params := page.CaptureScreenshot()
	params.Format = page.CaptureScreenshotFormatPng
	params.CaptureBeyondViewport = true
	params.FromSurface = true
	if requestParams.X != nil || requestParams.Y != nil || requestParams.Width != nil || requestParams.Height != nil {
		params.Clip = &page.Viewport{X: 0, Y: 0, Width: 1024.0, Height: 150.0, Scale: 1}
	}

	if requestParams.X != nil {
		params.Clip.X = float64(*requestParams.X)
	}

	if requestParams.Y != nil {
		params.Clip.Y = float64(*requestParams.Y)
	}

	if requestParams.Width != nil {
		params.Clip.Width = float64(*requestParams.Width)
	}

	if requestParams.Height != nil {
		params.Clip.Height = float64(*requestParams.Height)
	}

	if requestParams.Scale != nil {
		params.Clip.Scale = float64(*requestParams.Scale)
	}

	return params, nil
}
