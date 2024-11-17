package pdfire

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/cdproto/page"
)

// PaperFormats are the available paper formats.
var PaperFormats = map[string]struct {
	Width  float64
	Height float64
}{
	"letter": {
		Width:  8.5,
		Height: 11,
	},
	"legal": {
		Width:  8.5,
		Height: 14,
	},
	"tabloid": {
		Width:  11,
		Height: 17,
	},
	"ledger": {
		Width:  17,
		Height: 11,
	},
	"a0": {
		Width:  33.1,
		Height: 46.8,
	},
	"a1": {
		Width:  23.4,
		Height: 33.1,
	},
	"a2": {
		Width:  16.54,
		Height: 23.4,
	},
	"a3": {
		Width:  11.7,
		Height: 16.54,
	},
	"a4": {
		Width:  8.27,
		Height: 11.7,
	},
	"a5": {
		Width:  5.83,
		Height: 8.27,
	},
	"a6": {
		Width:  4.13,
		Height: 5.83,
	},
}

// UnitToPixels contains the to-pixel-ratios for the different units.
var UnitToPixels = map[string]float64{
	"px": 1,
	"in": 96,
	"cm": 37.8,
	"mm": 3.78,
}

var (
	// ErrInvalidJSON is a JSON syntax error.
	ErrInvalidJSON = errors.New("the json request is malformed")
	// ErrNoSource states that the options neither have an HTML string nor a URL.
	ErrNoSource = errors.New("no url or html provided")
)

var (
	// MediaScreen is the CSS "screen" media.
	MediaScreen = Media("screen")
	// MediaPrint is the CSS "print" media.
	MediaPrint = Media("print")
)

// ConversionOptions are the conversion options.
type ConversionOptions struct {
	HTML                   string
	URL                    string
	PDFParams              *page.PrintToPDFParams `json:"pdfParams"`
	ViewportWidth          int64
	ViewportHeight         int64
	BlockAds               bool
	Selector               string
	WaitForSelector        string
	WaitForSelectorTimeout time.Duration
	WaitUntil              string
	WaitUntilTimeout       time.Duration
	Delay                  time.Duration
	Timeout                time.Duration
	Headers                map[string]interface{}
	EmulateMedia           Media
	OwnerPassword          string
	UserPassword           string
	Watermark              *WatermarkConfig
}

// Media is a CSS media.
type Media string

// WatermarkConfig is the pdfcpu watermark configuration.
type WatermarkConfig struct {
	Query string
	OnTop bool
	Pages []string
}

// ParseError is returned when a PDF parameter cannot be parsed from a request body.
type ParseError struct {
	Key   string
	Value interface{}
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("Could not parse param \"%s\" (%v).", e.Key, e.Value)
}

// NewConversionOptions returns new converter options with default values.
func NewConversionOptions() *ConversionOptions {
	return &ConversionOptions{
		ViewportWidth:  1920,
		ViewportHeight: 1080,
		WaitUntil:      "load",
		Headers:        make(map[string]interface{}),
		EmulateMedia:   MediaScreen,
		PDFParams: &page.PrintToPDFParams{
			Scale:           1.0,
			PaperWidth:      8.5,
			PaperHeight:     11.0,
			MarginTop:       0.4,
			MarginBottom:    0.4,
			MarginRight:     0.4,
			MarginLeft:      0.4,
			PrintBackground: true,
			TransferMode:    page.PrintToPDFTransferModeReturnAsBase64,
		},
	}
}

// NewConversionOptionsFromJSONString returns new converter options from JSON.
func NewConversionOptionsFromJSONString(json string) (*ConversionOptions, error) {
	return NewConversionOptionsFromJSON(strings.NewReader(json))
}

// NewConversionOptionsFromJSON returns new converter options from JSON.
func NewConversionOptionsFromJSON(r io.Reader) (*ConversionOptions, error) {
	options := NewConversionOptions()
	params := options.PDFParams
	jsonMap := make(map[string]interface{})

	if err := json.NewDecoder(r).Decode(&jsonMap); err != nil {
		return nil, ErrInvalidJSON
	}

	html, err := parseString(jsonMap, "html", "")

	if err != nil {
		return nil, err
	}

	url, err := parseString(jsonMap, "url", "")

	if err != nil {
		return nil, err
	}

	landscape, err := parseBool(jsonMap, "landscape", false)

	if err != nil {
		return nil, err
	}

	displayHeaderFooter, err := parseBool(jsonMap, "displayHeaderFooter", false)

	if err != nil {
		return nil, err
	}

	printBackground, err := parseBool(jsonMap, "printBackground", true)

	if err != nil {
		return nil, err
	}

	scale, err := parseFloat64(jsonMap, "scale", 1.0)

	if err != nil {
		return nil, err
	}

	paperWidth, err := parseUnit(jsonMap, "paperWidth", options.PDFParams.PaperWidth)

	if err != nil {
		return nil, err
	}

	paperHeight, err := parseUnit(jsonMap, "paperHeight", options.PDFParams.PaperHeight)

	if err != nil {
		return nil, err
	}

	if format, err := parseString(jsonMap, "format", ""); err == nil {
		format = strings.ToLower(format)

		if f, ok := PaperFormats[format]; ok {
			paperWidth = f.Width
			paperHeight = f.Height
		}
	}

	marginTop, marginRight, marginBottom, marginLeft, err := parseMarginsFix(jsonMap)

	pageRanges, err := parseString(jsonMap, "pageRanges", "")

	if err != nil {
		return nil, err
	}

	headerTemplate, err := parseString(jsonMap, "headerTemplate", "")

	if err != nil {
		return nil, err
	}

	footerTemplate, err := parseString(jsonMap, "footerTemplate", "")

	if err != nil {
		return nil, err
	}

	preferCSSPageSize, err := parseBool(jsonMap, "preferCSSPageSize", false)

	if err != nil {
		return nil, err
	}

	viewportWidth, err := parseInt64(jsonMap, "viewportWidth", 1920)

	if err != nil {
		return nil, err
	}

	viewportHeight, err := parseInt64(jsonMap, "viewportHeight", 1080)

	if err != nil {
		return nil, err
	}

	blockAds, err := parseBool(jsonMap, "blockAds", false)

	if err != nil {
		return nil, err
	}

	selector, err := parseString(jsonMap, "selector", "")

	if err != nil {
		return nil, err
	}

	waitForSelector, err := parseString(jsonMap, "waitForSelector", "")

	if err != nil {
		return nil, err
	}

	waitForSelectorTimeout, err := parseDuration(jsonMap, "waitForSelectorTimeout", time.Duration(0))

	if err != nil {
		return nil, err
	}

	waitUntil, err := parseStringOnly(jsonMap, "waitUntil", "load", "load", "dom")

	if err != nil {
		return nil, err
	}

	waitUntilTimeout, err := parseDuration(jsonMap, "waitUntilTimeout", time.Duration(0))

	if err != nil {
		return nil, err
	}

	delay, err := parseDuration(jsonMap, "delay", time.Duration(0))

	if err != nil {
		return nil, err
	}

	timeout, err := parseDuration(jsonMap, "timeout", time.Duration(0))

	if err != nil {
		return nil, err
	}

	headers, err := parseHeaders(jsonMap)

	if err != nil {
		return nil, err
	}

	emulateMedia, err := parseEmulateMedia(jsonMap, MediaScreen)

	if err != nil {
		return nil, err
	}

	ownerPassword, err := parseString(jsonMap, "ownerPassword", "")

	if err != nil {
		return nil, err
	}

	userPassword, err := parseString(jsonMap, "userPassword", "")

	if err != nil {
		return nil, err
	}

	options.HTML = html
	options.URL = url
	params.Landscape = landscape
	params.DisplayHeaderFooter = displayHeaderFooter
	params.PrintBackground = printBackground
	params.Scale = scale
	params.PaperWidth = paperWidth
	params.PaperHeight = paperHeight
	params.MarginTop = marginTop
	params.MarginBottom = marginBottom
	params.MarginLeft = marginLeft
	params.MarginRight = marginRight
	params.PageRanges = pageRanges
	params.HeaderTemplate = headerTemplate
	params.FooterTemplate = footerTemplate
	params.PreferCSSPageSize = preferCSSPageSize
	options.ViewportWidth = viewportWidth
	options.ViewportHeight = viewportHeight
	options.BlockAds = blockAds
	options.Selector = selector
	options.WaitForSelector = waitForSelector
	options.WaitForSelectorTimeout = waitForSelectorTimeout
	options.WaitUntil = waitUntil
	options.WaitUntilTimeout = waitUntilTimeout
	options.Delay = delay
	options.Timeout = timeout
	options.Headers = headers
	options.EmulateMedia = emulateMedia
	options.OwnerPassword = ownerPassword
	options.UserPassword = userPassword

	return options, nil
}

func parseBool(jsonMap map[string]interface{}, key string, def bool) (bool, error) {
	value, ok := jsonMap[key]

	if !ok {
		return def, nil
	}

	v, ok := value.(bool)

	if !ok {
		return false, &ParseError{
			Key:   key,
			Value: value,
		}
	}

	return v, nil
}

func parseInt64(jsonMap map[string]interface{}, key string, def int64) (int64, error) {
	value, ok := jsonMap[key]

	if !ok {
		return def, nil
	}

	v, ok := value.(float64)
	uv := int64(v)

	if !ok {
		return 0, &ParseError{
			Key:   key,
			Value: value,
		}
	}

	return uv, nil
}

func parseFloat64(jsonMap map[string]interface{}, key string, def float64) (float64, error) {
	value, ok := jsonMap[key]

	if !ok {
		return def, nil
	}

	v, ok := value.(float64)

	if !ok {
		return 0, &ParseError{
			Key:   key,
			Value: value,
		}
	}

	return v, nil
}

func parseDuration(jsonMap map[string]interface{}, key string, def time.Duration) (time.Duration, error) {
	val, err := parseInt64(jsonMap, key, 0)

	if err != nil {
		return 0, err
	}

	if val < 0 {
		val = 0
	}

	return time.Duration(val) * time.Millisecond, nil
}

func parseString(jsonMap map[string]interface{}, key, def string) (string, error) {
	value, ok := jsonMap[key]

	if !ok {
		return def, nil
	}

	v, ok := value.(string)

	if !ok {
		return "", &ParseError{
			Key:   key,
			Value: value,
		}
	}

	return v, nil
}

func parseStrings(jsonMap map[string]interface{}, key string, def []string) ([]string, error) {
	raw, ok := jsonMap[key]

	if !ok {
		return def, nil
	}

	rvals, ok := raw.([]interface{})

	if !ok {
		return nil, &ParseError{
			Key:   key,
			Value: raw,
		}
	}

	vals := make([]string, 0)

	for _, rval := range rvals {
		val, ok := rval.(string)

		if !ok {
			return nil, &ParseError{
				Key:   key,
				Value: val,
			}
		}

		vals = append(vals, val)
	}

	return vals, nil
}

func parseStringOrStrings(jsonMap map[string]interface{}, key string, def []string) ([]string, error) {
	if vals, err := parseStrings(jsonMap, key, def); err == nil {
		return vals, err
	}

	raw, ok := jsonMap[key]

	if !ok {
		return def, nil
	}

	val, ok := raw.(string)

	if !ok {
		return nil, &ParseError{
			Key:   key,
			Value: raw,
		}
	}

	return []string{val}, nil
}

func parseStringOnly(jsonMap map[string]interface{}, key, def string, allowed ...string) (string, error) {
	param, err := parseString(jsonMap, key, def)

	if err != nil {
		return param, err
	}

	for _, a := range allowed {
		if a == param {
			return param, nil
		}
	}

	return def, &ParseError{
		Key:   key,
		Value: param,
	}
}

func parseUnit(jsonMap map[string]interface{}, key string, def float64) (float64, error) {
	raw, ok := jsonMap[key]

	if !ok {
		return def, nil
	}

	if fval, ok := raw.(float64); ok {
		return fval / float64(96), nil
	}

	sval, ok := raw.(string)

	if !ok {
		return 0, &ParseError{
			Key:   key,
			Value: sval,
		}
	}

	in, err := stringToInch(sval)

	if err != nil {
		return 0, &ParseError{
			Key:   key,
			Value: raw,
		}
	}

	return in, nil
}

func stringToInch(raw string) (float64, error) {
	if len(raw) < 2 {
		return 0, errors.New("invalid unit")
	}

	unit := strings.ToLower(raw[len(raw)-2:])
	valueText := ""
	unitPixels, ok := UnitToPixels[unit]

	if ok {
		valueText = raw[0 : len(raw)-2]
	} else {
		unit = "px"
		valueText = raw
	}

	expr := regexp.MustCompile("[a-zA-Z]+$")
	valueText = expr.ReplaceAllString(valueText, "")

	value, err := strconv.ParseFloat(valueText, 64)

	if err != nil {
		return 0, err
	}

	return pixelToInch(value * unitPixels), nil
}

func pixelToInch(pixel float64) float64 {
	return math.Round((pixel*100)/96) / 100
}

func parseMarginsFix(jsonMap map[string]interface{}) (float64, float64, float64, float64, error) {
	mt, mr, mb, ml, err := parseMargins(jsonMap)

	if err != nil {
		return mt, mr, mb, ml, err
	}

	vals := []*float64{
		&mt, &mr, &mb, &ml,
	}

	for _, v := range vals {
		if *v == 0 {
			*v = 0.00000001
		}
	}

	return mt, mr, mb, ml, err
}

func parseMargins(jsonMap map[string]interface{}) (float64, float64, float64, float64, error) {
	if margin, err := parseFloat64(jsonMap, "margin", -1); err == nil && margin > -1 {
		m := pixelToInch(margin)
		return m, m, m, m, nil
	}

	if margin, err := parseString(jsonMap, "margin", ""); err == nil && margin != "" {
		return parseMarginsFrom(margin)
	}

	var marginTop, marginRight, marginBottom, marginLeft float64

	marginTop, err := parseUnit(jsonMap, "marginTop", 0.4)

	if err != nil {
		return marginTop, marginRight, marginBottom, 0, err
	}

	marginRight, err = parseUnit(jsonMap, "marginRight", 0.4)

	if err != nil {
		return marginTop, marginRight, marginBottom, 0, err
	}

	marginBottom, err = parseUnit(jsonMap, "marginBottom", 0.4)

	if err != nil {
		return marginTop, marginRight, marginBottom, 0, err
	}

	marginLeft, err = parseUnit(jsonMap, "marginLeft", 0.4)

	if err != nil {
		return marginTop, marginRight, marginBottom, 0, err
	}

	return marginTop, marginRight, marginBottom, marginLeft, nil
}

func parseMarginsFrom(raw string) (float64, float64, float64, float64, error) {
	values := strings.Split(strings.Trim(raw, " "), " ")

	if len(values) == 0 {
		return 0, 0, 0, 0, &ParseError{
			Key:   "margin",
			Value: raw,
		}
	}

	var mt, mr, mb, ml float64

	mt, err := stringToInch(values[0])

	if err != nil {
		return 0, 0, 0, 0, err
	}

	if len(values) == 1 {
		return mt, mt, mt, mt, nil
	}

	mr, err = stringToInch(values[1])

	if err != nil {
		return 0, 0, 0, 0, err
	}

	if len(values) == 2 {
		return mt, mr, mt, mr, nil
	}

	mb, err = stringToInch(values[2])

	if err != nil {
		return 0, 0, 0, 0, err
	}

	if len(values) == 3 {
		return mt, mr, mb, mr, nil
	}

	ml, err = stringToInch(values[3])

	if err != nil {
		return 0, 0, 0, 0, err
	}

	return mt, mr, mb, ml, nil
}

func parseHeaders(jsonMap map[string]interface{}) (map[string]interface{}, error) {
	raw, ok := jsonMap["headers"]

	if !ok {
		return make(map[string]interface{}), nil
	}

	headers, ok := raw.(map[string]interface{})

	if !ok {
		return nil, &ParseError{
			Key:   "headers",
			Value: raw,
		}
	}

	return headers, nil
}

func parseEmulateMedia(jsonMap map[string]interface{}, def Media) (Media, error) {
	raw, ok := jsonMap["emulateMedia"]

	if !ok {
		return def, nil
	}

	val, ok := raw.(string)

	if !ok {
		return def, &ParseError{
			Key:   "emulateMedia",
			Value: raw,
		}
	}

	media := Media(val)

	if media != MediaScreen && media != MediaPrint {
		return def, &ParseError{
			Key:   "emulateMedia",
			Value: media,
		}
	}

	return media, nil
}
