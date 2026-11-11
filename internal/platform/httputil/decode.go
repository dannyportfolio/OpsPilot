package httputil

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// DefaultMaxBodyBytes is the standard 1 MB maximum payload limit across all API endpoints.
const DefaultMaxBodyBytes = 1 << 20 // 1,048,576 bytes

// DecodeJSON strictly decodes incoming JSON payloads into the destination struct.
// It enforces body size bounding via http.MaxBytesReader, rejects unknown JSON fields,
// verifies payload existence, and ensures no trailing or multiple JSON tokens exist.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any, maxBytes int64) error {
	if r.Body == nil {
		return errors.New("request body is empty")
	}
	defer r.Body.Close()

	if maxBytes <= 0 {
		maxBytes = DefaultMaxBodyBytes
	}

	limitedReader := http.MaxBytesReader(w, r.Body, maxBytes)
	decoder := json.NewDecoder(limitedReader)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return err
	}

	// Verify no additional JSON values follow
	var extra json.RawMessage
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("request body contains multiple JSON values or extra data")
	}

	return nil
}
