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

	printOptions, err := getScreenshotOptions(pngRequestParams)
	if err != nil {
		log.Printf("Error: %s", err.Error())
		return nil, err
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

	var base64EncodedData string
	match, _ := regexp.MatchString("(?i)^(https?|file|data):", pngRequestParams.Data)
	if match {
		base64EncodedData = pngRequestParams.Data
	} else {
		base64EncodedData = "data:text/html;base64," + base64.StdEncoding.EncodeToString([]byte(pngRequestParams.Data))
	}

	var screenshotBuffer []byte
	err = chromedp.Run(ctx,
		chromedp.Navigate(base64EncodedData),
		printToPng(&screenshotBuffer, printOptions),
	)
	if err != nil {
		return nil, err
	}

	// chromedp.Run(ctx, printToPng(base64EncodedData, printOptions, cancel, &screenshotBuffer))
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

	log.Println("SCREENSHOT - 1")
	return chromedp.ActionFunc(func(ctx context.Context) error {
		log.Println("SCREENSHOT - 2")
		buf, err := params.Do(ctx)
		// page.CaptureScreenshot().
		// 	WithFormat(params.Format).
		// 	WithCaptureBeyondViewport(true).
		// 	WithFromSurface(true).
		// 	WithClip(params.Clip).
		// 	Do(ctx)
		log.Println("SCREENSHOT - 3")
		*res = buf
		// log.Printf("Done")
		return err
	})
}

// func printToPng(base64EncodedData string, params *page.CaptureScreenshotParams, cancelContext context.CancelFunc, screenshotBuffer *[]byte) chromedp.Tasks {
// 	log.Println("Execute screenshot 1")

// 	return chromedp.Tasks{
// 		chromedp.Navigate(base64EncodedData),
// 		chromedp.ActionFunc(func(ctx context.Context) error {
// 			log.Println("Execute screenshot 2")
// 			defer cancelContext()
// 			buf, err := params.Do(ctx)
// 			if err != nil {
// 				log.Println("Got Error")
// 				return err
// 			}
// 			log.Println("No Error")
// 			log.Printf("Buffer Length %d", len(buf))

// 			*screenshotBuffer = buf
// 			return nil
// 		}),
// 	}
// }

func getScreenshotOptions(requestParams *PngRequest) (*page.CaptureScreenshotParams, error) {
	params := page.CaptureScreenshot()
	params.Format = page.CaptureScreenshotFormatPng
	params.Clip = &page.Viewport{X: 0, Y: 0, Width: 1020.0, Height: 150.0}

	if requestParams.Width != nil {
		params.Clip.Width = float64(*requestParams.Width)
	}

	if requestParams.Height != nil {
		params.Clip.Height = float64(*requestParams.Height)
	}

	return params, nil
}
