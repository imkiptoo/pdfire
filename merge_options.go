package pdfire

import (
	"encoding/json"
	"io"
	"strings"
)

// MergeOptions are the merge options.
type MergeOptions struct {
	Documents     []*ConversionOptions
	OwnerPassword string
	UserPassword  string
	Watermark     *WatermarkConfig
}

// NewMergeOptions returns new merge options.
func NewMergeOptions() *MergeOptions {
	return &MergeOptions{
		Documents: make([]*ConversionOptions, 0),
	}
}

// NewMergeOptionsFromJSONString returns new merge options from JSON.
func NewMergeOptionsFromJSONString(json string) (*MergeOptions, error) {
	return NewMergeOptionsFromJSON(strings.NewReader(json))
}

// NewMergeOptionsFromJSON returns new merge options from JSON.
func NewMergeOptionsFromJSON(r io.Reader) (*MergeOptions, error) {
	jsonMap := make(map[string]interface{})

	if err := json.NewDecoder(r).Decode(&jsonMap); err != nil {
		return nil, ErrInvalidJSON
	}

	data, ok := jsonMap["documents"]

	if !ok {
		return nil, &ParseError{
			Key: "documents",
		}
	}

	docdata, ok := data.([]interface{})

	if !ok {
		return nil, &ParseError{
			Key:   "documents",
			Value: data,
		}
	}

	docoptions := make([]*ConversionOptions, 0)

	for _, data := range docdata {
		jsn, err := json.Marshal(data)

		if err != nil {
			return nil, err
		}

		options, err := NewConversionOptionsFromJSONString(string(jsn))

		if err != nil {
			return nil, err
		}

		options.OwnerPassword = ""
		options.UserPassword = ""
		docoptions = append(docoptions, options)
	}

	ownerPassword, err := parseString(jsonMap, "ownerPassword", "")

	if err != nil {
		return nil, err
	}

	userPassword, err := parseString(jsonMap, "userPassword", "")

	if err != nil {
		return nil, err
	}

	return &MergeOptions{
		Documents:     docoptions,
		OwnerPassword: ownerPassword,
		UserPassword:  userPassword,
	}, nil
}
