package uploads

import "time"

type Upload struct {
	ID           string
	AuthorID     string
	FilePath     string
	OriginalName string
	MimeType     string
	SizeBytes    int64
	CreatedAt    time.Time
}
