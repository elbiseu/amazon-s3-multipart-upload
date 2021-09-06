package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"strings"
)

const (
	minUploadPartSize int   = 1024 * 1024 * 5    // 5 MB
	maxContentSize    int64 = 1024 * 1024 * 2500 // 2500 MB
)

func FileHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		contentType := r.Header.Get("Content-Type")

		if !(strings.HasPrefix(contentType, "image/") || strings.HasPrefix(contentType, "video/")) {
			w.WriteHeader(http.StatusUnsupportedMediaType)
			return
		}

		if r.ContentLength > maxContentSize {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			return
		}

		buffer := bytes.Buffer{}
		lastPart := false
		partNumber := 1 // The first part number must always start with 1.

		for !lastPart {
			n, err := io.CopyN(&buffer, r.Body, 4096)

			if n == 0 || err == io.EOF {
				lastPart = true
			} else if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			if buffer.Len() > minUploadPartSize || lastPart {
				buffer.Reset() // The buffer is empty to the next parts.
				partNumber++
			}
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func main() {
	serveMux := http.NewServeMux()

	serveMux.HandleFunc("/api/v1/file", FileHandler)

	if err := http.ListenAndServe(":8080", serveMux); err != nil {
		log.Fatalln(err)
	}
}
