package pdfire_test

import (
	"bytes"
	"context"
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/modernice/pdfire"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	flag.Parse()
	if testing.Short() {
		return
	}

	os.Exit(m.Run())
}

func TestConvertHTML(t *testing.T) {
	assert := assert.New(t)
	wd, _ := os.Getwd()
	file, _ := os.Open(filepath.Join(wd, "testdata/html.html"))
	defer file.Close()

	pdf := bytes.NewBuffer(make([]byte, 0))
	options := pdfire.NewConversionOptions()
	err := pdfire.Convert(context.Background(), pdf, options)

	assert.Nil(err)

	b := make([]byte, 0)
	file.Read(b)

	if pdf.Len() < len(b) {
		t.Error("Generated PDF is smaller than the provided HTML.")
	}
}

func TestConvertURL(t *testing.T) {
	assert := assert.New(t)
	wd, _ := os.Getwd()
	filepath := filepath.Join(wd, "testdata/html.html")
	html, _ := ioutil.ReadFile(filepath)

	pdf := bytes.NewBuffer(make([]byte, 0))
	options := pdfire.NewConversionOptions()
	err := pdfire.Convert(context.Background(), pdf, options)

	assert.Nil(err)

	if pdf.Len() < len(html) {
		t.Error("Generated PDF is smaller than the provided HTML.")
	}
}
