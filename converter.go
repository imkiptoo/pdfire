package pdfire

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/google/uuid"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

var (
	// ErrTimeout is returned when the conversion times out.
	ErrTimeout = errors.New("conversion timed out")
	// ErrWaitUntilTimeout is returned when the Chrome DevTools times out while waiting for the "load" or "DOMContentLoaded" event.
	ErrWaitUntilTimeout = errors.New("WaitUntil timed out")
	// ErrNoBody is returned when the page has no 'body' element.
	ErrNoBody = errors.New("page has no 'body' element")
)

type result struct {
	index int
	buf   *bytes.Buffer
}

// Convert creates a PDF from the given options.
func Convert(ctx context.Context, w io.Writer, options *ConversionOptions) error {
	if options.URL != "" {
		return ConvertURL(ctx, w, options)
	}

	return ConvertHTML(ctx, w, options)
}

// ConvertHTML creates a PDF from an HTML string.
func ConvertHTML(ctx context.Context, w io.Writer, options *ConversionOptions) error {
	ctx, cancel := conversionContext(ctx, options)
	defer cancel()

	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()

	id := uuid.New()
	r := strings.NewReader(options.HTML)
	file, err := createAndCloseHTMLFile(id, r)

	if err != nil {
		return err
	}

	beforeNavAction, waiter := beforeNavigation(options)
	buf := bytes.NewBuffer([]byte{})

	if err := chromedp.Run(
		ctx,
		beforeNavAction,
		chromedp.Navigate(fmt.Sprintf("file://%s", file.Name())),
		afterNavigation(options, waiter),
		printToPDFAction(buf, options),
	); err != nil {
		if err == context.DeadlineExceeded {
			return ErrTimeout
		}

		return err
	}

	if err := os.Remove(file.Name()); err != nil {
		return err
	}

	if options.Watermark != nil {
		if buf, err = watermark(buf, options.Watermark); err != nil {
			return err
		}
	}

	buf, err = secure(buf, options.OwnerPassword, options.UserPassword)

	if err != nil {
		return err
	}

	_, err = io.Copy(w, buf)

	return err
}

// ConvertURL creates a PDF from a URL.
func ConvertURL(ctx context.Context, w io.Writer, options *ConversionOptions) error {
	ctx, cancel := conversionContext(ctx, options)
	defer cancel()

	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()

	beforeNavAction, waiter := beforeNavigation(options)
	buf := bytes.NewBuffer([]byte{})

	if err := chromedp.Run(
		ctx,
		beforeNavAction,
		chromedp.Navigate(options.URL),
		afterNavigation(options, waiter),
		printToPDFAction(buf, options),
	); err != nil {
		if err == context.DeadlineExceeded {
			return ErrTimeout
		}

		return err
	}

	var err error

	if options.Watermark != nil {
		if buf, err = watermark(buf, options.Watermark); err != nil {
			return err
		}
	}

	buf, err = secure(buf, options.OwnerPassword, options.UserPassword)

	if err != nil {
		return err
	}

	_, err = io.Copy(w, buf)

	return err
}

// Merge creates multiple PDFs and merges them together into a single file.
func Merge(ctx context.Context, w io.Writer, options *MergeOptions) error {
	for _, convopt := range options.Documents {
		convopt.OwnerPassword = ""
		convopt.UserPassword = ""
	}

	cres := make(chan result, len(options.Documents))
	cerr := make(chan error, len(options.Documents))

	for i, convopt := range options.Documents {
		go forMerge(ctx, i, convopt, cres, cerr)
	}

	err := mergeDocs(ctx, w, options, cres, cerr)

	if err != nil {
		return err
	}

	return nil
}

func forMerge(ctx context.Context, index int, options *ConversionOptions, cres chan<- result, cerr chan<- error) {
	buf := bytes.NewBuffer([]byte{})

	if err := Convert(ctx, buf, options); err != nil {
		cerr <- err
	}

	cres <- result{
		index: index,
		buf:   buf,
	}
}

func mergeDocs(ctx context.Context, w io.Writer, options *MergeOptions, cres <-chan result, cerrs <-chan error) error {
	bufs := make([]*bytes.Buffer, cap(cres))
	c := 0

	for {
		if c == len(bufs) {
			break
		}

		select {
		case err := <-cerrs:
			return err
		case res := <-cres:
			bufs[res.index] = res.buf
			c++
		case <-ctx.Done():
			return ErrTimeout
		}
	}

	readers := make([]io.ReadSeeker, len(bufs))

	for i, buf := range bufs {
		readers[i] = bytes.NewReader(buf.Bytes())
	}

	merged := bytes.NewBuffer([]byte{})
	if err := api.Merge(readers, merged, nil); err != nil {
		return err
	}

	b, err := secure(merged, options.OwnerPassword, options.UserPassword)

	if err != nil {
		return err
	}

	_, err = io.Copy(w, b)

	return err
}

func conversionContext(ctx context.Context, options *ConversionOptions) (context.Context, context.CancelFunc) {
	var cancel context.CancelFunc

	if options.Timeout == 0 {
		ctx, cancel = context.WithCancel(ctx)
	} else {
		ctx, cancel = context.WithTimeout(ctx, options.Timeout)
	}

	return ctx, cancel
}

func createAndCloseHTMLFile(id uuid.UUID, r io.Reader) (*os.File, error) {
	os.MkdirAll(filepath.Join(os.TempDir(), "pdfire/tmp/html"), os.ModePerm)
	file, err := os.Create(filepath.Join(os.TempDir(), fmt.Sprintf("pdfire/tmp/html/%s.html", id.String())))

	if err != nil {
		return nil, err
	}

	defer file.Close()
	_, err = io.Copy(file, r)

	return file, nil
}

func beforeNavigation(options *ConversionOptions) (chromedp.ActionFunc, <-chan bool) {
	waiter := make(chan bool, 1)

	return func(ctx context.Context) error {
		if err := emulation.SetDeviceMetricsOverride(options.ViewportWidth, options.ViewportHeight, 1, false).Do(ctx); err != nil {
			return err
		}

		if err := page.SetAdBlockingEnabled(options.BlockAds).Do(ctx); err != nil {
			return err
		}

		if err := network.Enable().Do(ctx); err != nil {
			return err
		}

		if err := network.SetExtraHTTPHeaders(options.Headers).Do(ctx); err != nil {
			return err
		}

		if err := emulation.SetEmulatedMedia().WithMedia(string(options.EmulateMedia)).Do(ctx); err != nil {
			return err
		}

		chromedp.ListenTarget(ctx, func(ev interface{}) {
			switch ev.(type) {
			case *page.EventLoadEventFired:
				if options.WaitUntil == "load" {
					waiter <- true
				}
			case *page.EventDomContentEventFired:
				if options.WaitUntil == "dom" {
					waiter <- true
				}
			}
		})

		return nil
	}, waiter
}

func afterNavigation(options *ConversionOptions, waiter <-chan bool) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		if options.WaitForSelector != "" {
			var waitCtx context.Context
			var cancel context.CancelFunc

			if options.WaitForSelectorTimeout > 0 {
				waitCtx, cancel = context.WithCancel(ctx)
			} else {
				waitCtx, cancel = context.WithTimeout(ctx, time.Duration(options.WaitForSelectorTimeout)*time.Millisecond)
			}

			defer cancel()

			if err := chromedp.WaitReady(options.WaitForSelector).Do(waitCtx); err != nil {
				return err
			}
		}

		if options.WaitUntilTimeout > 0 {
			if !<-waiterTimeout(waiter, time.Duration(options.WaitUntilTimeout)*time.Millisecond) {
				return ErrWaitUntilTimeout
			}
		} else {
			<-waiter
		}

		if options.Delay > 0 {
			<-time.After(options.Delay)
		}

		if options.Selector != "" {
			htmlb := strings.Builder{}
			htmlb.WriteString("<body>")

			var elhtml string
			if err := chromedp.OuterHTML(options.Selector, &elhtml).Do(ctx); err != nil {
				return err
			}

			htmlb.WriteString(elhtml)
			htmlb.WriteString("</body>")

			var nodes []*cdp.Node
			if err := chromedp.Nodes("body", &nodes, chromedp.ByQuery).Do(ctx); err != nil || len(nodes) == 0 {
				return err
			}

			body := nodes[0]

			if err := dom.SetOuterHTML(body.NodeID, htmlb.String()).Do(ctx); err != nil {
				return err
			}
		}

		return nil
	}
}

func waiterTimeout(waiter <-chan bool, d time.Duration) <-chan bool {
	towaiter := make(chan bool)

	go func() {
		select {
		case <-waiter:
			towaiter <- true
		case <-time.NewTimer(d).C:
			towaiter <- false
		}
	}()

	return towaiter
}

func printToPDFAction(w io.Writer, options *ConversionOptions) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		data, _, err := options.PDFParams.Do(ctx)

		if err != nil {
			return err
		}

		_, err = w.Write(data)

		return err
	}
}

func secure(buf *bytes.Buffer, ownerPw, userPw string) (*bytes.Buffer, error) {
	if ownerPw == "" && userPw == "" {
		return buf, nil
	}

	cfg := pdfcpu.NewAESConfiguration(userPw, ownerPw, 256)
	final := bytes.NewBuffer([]byte{})

	cfg.Cmd = pdfcpu.ENCRYPT

	if err := api.Optimize(bytes.NewReader(buf.Bytes()), final, cfg); err != nil {
		return nil, err
	}

	return final, nil
}

func watermark(buf *bytes.Buffer, config *WatermarkConfig) (*bytes.Buffer, error) {
	wm, err := pdfcpu.ParseWatermarkDetails(config.Query, config.OnTop)

	if err != nil {
		return nil, err
	}

	w := bytes.NewBuffer([]byte{})

	if err := api.AddWatermarks(bytes.NewReader(buf.Bytes()), w, config.Pages, wm, nil); err != nil {
		return nil, err
	}

	return w, nil
}
