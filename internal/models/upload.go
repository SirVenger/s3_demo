package models

type UploadResult struct {
	FileID string
	Size   int64
	Parts  int
}

type ChunkPlan struct {
	Total int
	Size  int64
}
