package pdfire_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/modernice/pdfire"
	"github.com/chromedp/cdproto/page"
	"github.com/stretchr/testify/assert"
)

func TestNewConversionOptions(t *testing.T) {
	assert := assert.New(t)
	options := pdfire.NewConversionOptions()

	assert.Equal("", options.HTML)
	assert.Equal("", options.URL)
	assert.Equal(false, options.PDFParams.Landscape)
	assert.Equal(false, options.PDFParams.DisplayHeaderFooter)
	assert.Equal(true, options.PDFParams.PrintBackground)
	assert.Equal(1.0, options.PDFParams.Scale)
	assert.Equal(8.5, options.PDFParams.PaperWidth)
	assert.Equal(11.0, options.PDFParams.PaperHeight)
	assert.Equal(0.4, options.PDFParams.MarginTop)
	assert.Equal(0.4, options.PDFParams.MarginBottom)
	assert.Equal(0.4, options.PDFParams.MarginLeft)
	assert.Equal(0.4, options.PDFParams.MarginRight)
	assert.Equal("", options.PDFParams.PageRanges)
	assert.Equal(false, options.PDFParams.IgnoreInvalidPageRanges)
	assert.Equal("", options.PDFParams.HeaderTemplate)
	assert.Equal("", options.PDFParams.FooterTemplate)
	assert.Equal(false, options.PDFParams.PreferCSSPageSize)
	assert.Equal(page.PrintToPDFTransferModeReturnAsBase64, options.PDFParams.TransferMode)
	assert.Equal(int64(1920), options.ViewportWidth)
	assert.Equal(int64(1080), options.ViewportHeight)
	assert.Equal(false, options.BlockAds)
	assert.Equal("", options.Selector)
	assert.Equal("", options.WaitForSelector)
	assert.Equal(time.Duration(0), options.WaitForSelectorTimeout)
	assert.Equal("load", options.WaitUntil)
	assert.Equal(time.Duration(0), options.WaitUntilTimeout)
	assert.Equal(time.Duration(0), options.Delay)
	assert.Equal(time.Duration(0), options.Timeout)
	assert.IsType(map[string]interface{}{}, options.Headers)
	assert.Equal(pdfire.MediaScreen, options.EmulateMedia)
	assert.Equal("", options.OwnerPassword)
	assert.Equal("", options.UserPassword)
}

func TestNewConversionOptionsFromJSON(t *testing.T) {
	assert := assert.New(t)
	wd, _ := os.Getwd()
	filepath := filepath.Join(wd, "testdata/conversion.json")
	reader, _ := os.Open(filepath)
	defer reader.Close()

	options, err := pdfire.NewConversionOptionsFromJSON(reader)

	assert.Nil(err)

	assert.Equal("<p>This is a text.</p>", options.HTML)
	assert.Equal("http://localhost:3000/test", options.URL)
	assert.Equal(true, options.PDFParams.Landscape)
	assert.Equal(true, options.PDFParams.DisplayHeaderFooter)
	assert.Equal(true, options.PDFParams.PrintBackground)
	assert.Equal(1.4, options.PDFParams.Scale)
	assert.Equal(10.5, options.PDFParams.PaperWidth)
	assert.Equal(12.0, options.PDFParams.PaperHeight)
	assert.Equal(0.5, options.PDFParams.MarginTop)
	assert.Equal(0.4, options.PDFParams.MarginBottom)
	assert.Equal(0.3, options.PDFParams.MarginLeft)
	assert.Equal(0.7, options.PDFParams.MarginRight)
	assert.Equal("1-3", options.PDFParams.PageRanges)
	assert.Equal("<p>HEADER</p>", options.PDFParams.HeaderTemplate)
	assert.Equal("<p>FOOTER</p>", options.PDFParams.FooterTemplate)
	assert.Equal(true, options.PDFParams.PreferCSSPageSize)
	assert.Equal(page.PrintToPDFTransferModeReturnAsBase64, options.PDFParams.TransferMode)
	assert.Equal(int64(1280), options.ViewportWidth)
	assert.Equal(int64(720), options.ViewportHeight)
	assert.Equal(true, options.BlockAds)
	assert.Equal("#pdf", options.Selector)
	assert.Equal("#wait-selector", options.WaitForSelector)
	assert.Equal(time.Duration(3000)*time.Millisecond, options.WaitForSelectorTimeout)
	assert.Equal("dom", options.WaitUntil)
	assert.Equal(time.Duration(10000)*time.Millisecond, options.WaitUntilTimeout)
	assert.Equal(time.Duration(2000)*time.Millisecond, options.Delay)
	assert.Equal(time.Duration(60000)*time.Millisecond, options.Timeout)
	assert.Equal("test-header-value1", options.Headers["test-header-key1"])
	assert.Equal("test-header-value2", options.Headers["test-header-key2"])
	assert.Equal(pdfire.MediaPrint, options.EmulateMedia)
	assert.Equal("ownerpw", options.OwnerPassword)
	assert.Equal("userpw", options.UserPassword)
}

func TestNewConversionOptionsFromJSONInvalid(t *testing.T) {
	assert := assert.New(t)
	wd, _ := os.Getwd()
	filepath := filepath.Join(wd, "testdata/invalid-conversion.json")
	reader, _ := os.Open(filepath)
	defer reader.Close()

	options, err := pdfire.NewConversionOptionsFromJSON(reader)

	assert.Nil(options)
	assert.IsType(&pdfire.ParseError{}, err)
}
