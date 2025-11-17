package models

// UploadResult возвращается после успешной загрузки и содержит ключевые метаданные.
type UploadResult struct {
	FileID string
	Size   int64
	Parts  int
}

// ChunkPlan описывает, на сколько частей нужно разбить файл и какого они размера.
type ChunkPlan struct {
	Total int
	Size  int64
}
