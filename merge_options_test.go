package pdfire_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/modernice/pdfire"
	"github.com/stretchr/testify/assert"
)

func TestNewMergeOptions(t *testing.T) {
	assert := assert.New(t)

	options := pdfire.NewMergeOptions()

	assert.IsType([]*pdfire.ConversionOptions{}, options.Documents)
	assert.Equal("", options.OwnerPassword)
	assert.Equal("", options.UserPassword)
}

func TestNewMergeOptionsFromJSON(t *testing.T) {
	assert := assert.New(t)

	wd, _ := os.Getwd()
	path := filepath.Join(wd, "testdata/merge.json")
	reader, _ := os.Open(path)
	options, err := pdfire.NewMergeOptionsFromJSON(reader)

	assert.Nil(err)
	assert.Len(options.Documents, 3)
	assert.Equal(options.Documents[0].HTML, "<p>Page 1</p>")
	assert.Equal(options.Documents[1].HTML, "<p>Page 2</p>")
	assert.Equal(options.Documents[2].HTML, "<p>Page 3</p>")
	assert.Equal("", options.Documents[1].OwnerPassword)
	assert.Equal("owner-pw", options.OwnerPassword)
	assert.Equal("user-pw", options.UserPassword)
}
