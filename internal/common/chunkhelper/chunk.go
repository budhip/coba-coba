package chunkhelper

import (
	"context"
	"time"

	xlog "bitbucket.org/Amartha/go-x/log"
)

// DownloadProgress tracks download progress for monitoring
type DownloadProgress struct {
	SummaryID   string
	StartTime   time.Time
	TotalChunks int
	TotalRows   int
	LastChunkAt time.Time
}

func (p *DownloadProgress) LogProgress(ctx context.Context, chunkNumber int, chunkSize int) {
	elapsed := time.Since(p.StartTime)
	avgTimePerChunk := elapsed / time.Duration(chunkNumber)

	xlog.Info(ctx, "[DOWNLOAD-PROGRESS]",
		xlog.String("summary_id", p.SummaryID),
		xlog.Int("chunk_number", chunkNumber),
		xlog.Int("chunk_size", chunkSize),
		xlog.Int("total_rows", p.TotalRows),
		xlog.String("elapsed", elapsed.String()),
		xlog.String("avg_per_chunk", avgTimePerChunk.String()))
}
