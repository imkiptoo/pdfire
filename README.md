# PDFire - HTML to PDF converter

> Convert your HTML pages to beautiful PDF files. Supports the latest HTML, CSS and Javascript features.

This tool creates PDF files through the Chrome DevTools Protocol using the [chromedp](https://github.com/chromedp/chromedp) & [pdfcpu](https://github.com/pdfcpu/pdfcpu) package.

## Installation

```sh
go get github.com/imkiptoo/pdfire
```

## Usage

### Pre-configured server

```go
package main

import (
    "net/http"

    "github.com/imkiptoo/pdfire/server"
)

func main() {
    if err := http.ListenAndServe("localhost:3000", server.New()); err != nil {
      panic(err)
    }
}
```

```sh
curl -X POST localhost:3000/conversions \n
    -H "Content-Type: application/json" \n
    -d '{"url": "https://google.com"}'
```

### Manual use

```go
package main

import (
    "bytes"
    "strings"
    "github.com/imkiptoo/pdfire"
)

func main() {
    pdf := bytes.NewBuffer(make([]byte, 0))
    options := pdfire.NewOptions()

    // Create PDF from an URL
    err := pdfire.URLToPDF("https://google.com", pdf, options)

    // Create PDF from an HTML string
    err := pdfire.HTMLToPDF(strings.NewReader("<p>Example paragraph</p>"), pdf, options)

    // Create PDF from an HTML file
    err := pdfire.FileToPDF("/path/to/file", pdf, options)
}
```

