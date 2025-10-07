package repositories

import (
	"bufio"
	"context"
	"encoding/csv"
	"io"
	"mime/multipart"
)

type FileRepository interface {
	StreamReadMultipartFile(ctx context.Context, file *multipart.FileHeader) <-chan StreamReadMultipartFileResult
	StreamReadCSVFile(ctx context.Context, fileRead io.ReadCloser) <-chan StreamReadCSVFileResult
}

type fileRepo struct{}

// StreamReadMultipartFile implements FileRepository.
func (*fileRepo) StreamReadMultipartFile(ctx context.Context, file *multipart.FileHeader) <-chan StreamReadMultipartFileResult {
	resultCh := make(chan StreamReadMultipartFileResult)

	go func() {
		defer close(resultCh)

		// Open file
		openedFile, err := file.Open()
		if err != nil {
			resultCh <- StreamReadMultipartFileResult{Err: err}
			return
		}

		// Scanner
		scanner := bufio.NewScanner(openedFile)
		bufferSize := 1024 * 512
		buffer := make([]byte, bufferSize)
		scanner.Buffer(buffer, bufferSize)

		// Read
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
				resultCh <- StreamReadMultipartFileResult{Data: scanner.Text()}
			}
		}

		if err := scanner.Err(); err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				resultCh <- StreamReadMultipartFileResult{Err: err}
			}
		}

		// Close
		openedFile.Close()
	}()

	return resultCh
}

// StreamReadCSVFile reads a CSV file from a local path and streams each row.
func (*fileRepo) StreamReadCSVFile(ctx context.Context, fileRead io.ReadCloser) <-chan StreamReadCSVFileResult {
	resultCh := make(chan StreamReadCSVFileResult)

	go func() {
		defer close(resultCh)

		// Create a buffered reader
		reader := bufio.NewReader(fileRead)
		csvReader := csv.NewReader(reader)

		// Read rows
		for {
			select {
			case <-ctx.Done():
				return
			default:
				// Read a single record
				row, err := csvReader.Read()
				if err != nil {
					if err.Error() == "EOF" {
						return
					}
					resultCh <- StreamReadCSVFileResult{Err: err}
					return
				}

				resultCh <- StreamReadCSVFileResult{Data: row}
			}
		}
	}()

	return resultCh
}

func NewFileRepository() FileRepository {
	return &fileRepo{}
}

type (
	StreamReadMultipartFileResult struct {
		Data string
		Err  error
	}

	StreamReadCSVFileResult struct {
		Data []string
		Err  error
	}
)
