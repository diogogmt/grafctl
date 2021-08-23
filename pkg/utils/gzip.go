package utils

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
)

func Gzip(ctx context.Context, by []byte) ([]byte, error) {
	var gzipBuffer bytes.Buffer
	gzipWriter, err := gzip.NewWriterLevel(&gzipBuffer, gzip.BestCompression)
	if err != nil {
		return nil, fmt.Errorf("gzip.NewWriterLevel: %w", err)
	}
	if _, err := gzipWriter.Write(by); err != nil {
		return nil, fmt.Errorf("gzipWriter.Write: %w", err)
	}
	if err := gzipWriter.Close(); err != nil {
		return nil, fmt.Errorf("gzipWriter.Close: %w", err)
	}
	return gzipBuffer.Bytes(), nil
}
