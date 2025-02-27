package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"os"
	"regexp"
	"fmt"
	"math"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
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
	DomId    string   `json:"domId" form:"domId"`
	Xpath    string   `json:"xpath" form:"xpath"`
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
		printToPng(&screenshotBuffer, printOptions, pngRequestParams),
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

func printToPng(res *[]byte, params *page.CaptureScreenshotParams, requestParams *PngRequest) chromedp.Action {
	if res == nil {
		panic("res cannot be nil")
	}

	if requestParams.DomId {

	}

	if requestParams.Xpath {

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

	if (requestParams.DomId != nil && requestParams.Xpath != nil) || requestParams.X != nil || requestParams.Y != nil || requestParams.Width != nil || requestParams.Height != nil {
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

func screenshotElement(sel interface{}, params *page.CaptureScreenshotParams, res *[]byte, opts ...chromedp.QueryOption) chromedp.QueryAction {
	if res == nil {
		panic("picbuf cannot be nil")
	}

	return chromedp.QueryAfter(sel, func(ctx context.Context, execCtx runtime.ExecutionContextID, nodes ...*cdp.Node) error {
		if len(nodes) < 1 {
			return fmt.Errorf("selector %q did not return any nodes", sel)
		}
		return screenshotNodes(nodes, params, res).Do(ctx)
	}, append(opts, chromedp.NodeVisible)...)
}

// ScreenshotNodes is an action that captures/takes a screenshot of the
// specified nodes, by calculating the extents of the top most left node and
// bottom most right node.
func screenshotNodes(nodes []*cdp.Node, params *page.CaptureScreenshotParams, res *[]byte) chromedp.Action {
	if len(nodes) == 0 {
		panic("nodes must be non-empty")
	}
	if res == nil {
		panic("res cannot be nil")
	}

	return chromedp.ActionFunc(func(ctx context.Context) error {
		var clip page.Viewport

		// get box model of first node
		if err := callFunctionOnNode(ctx, nodes[0], getClientRectJS, &clip); err != nil {
			return err
		}

		// remainder
		for _, node := range nodes[1:] {
			var v page.Viewport
			// get box model of first node
			if err := callFunctionOnNode(ctx, node, getClientRectJS, &v); err != nil {
				return err
			}
			clip.X, clip.Width = extents(clip.X, clip.Width, v.X, v.Width)
			clip.Y, clip.Height = extents(clip.Y, clip.Height, v.Y, v.Height)
		}

		// The "Capture node screenshot" command does not handle fractional dimensions properly.
		// Let's align with puppeteer:
		// https://github.com/puppeteer/puppeteer/blob/bba3f41286908ced8f03faf98242d4c3359a5efc/src/common/Page.ts#L2002-L2011
		x, y := math.Round(clip.X), math.Round(clip.Y)
		clip.Width, clip.Height = math.Round(clip.Width+clip.X-x), math.Round(clip.Height+clip.Y-y)
		clip.X, clip.Y = x, y

		// take screenshot of the box
		buf, err := params.WithClip(&clip).Do(ctx)

		if err != nil {
			return err
		}

		*res = buf
		return nil
	})
}

func callFunctionOnNode(ctx context.Context, node *cdp.Node, function string, res interface{}, args ...interface{}) error {
	r, err := dom.ResolveNode().WithNodeID(node.NodeID).Do(ctx)
	if err != nil {
		return err
	}
	err = chromedp.CallFunctionOn(function, res,
		func(p *runtime.CallFunctionOnParams) *runtime.CallFunctionOnParams {
			return p.WithObjectID(r.ObjectID)
		},
		args...,
	).Do(ctx)

	if err != nil {
		return err
	}

	// Try to release the remote object.
	// It will fail if the page is navigated or closed,
	// and it's okay to ignore the error in this case.
	_ = runtime.ReleaseObject(r.ObjectID).Do(ctx)

	return nil
}

func extents(m, n, o, p float64) (float64, float64) {
	a := min(m, o)
	b := max(m+n, o+p)
	return a, b - a
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
