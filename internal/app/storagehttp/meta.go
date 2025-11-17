package storagehttp

import (
	"encoding/json"
	"os"
)

// partMeta описывает одну часть, сохранённую storage-сервером.
type partMeta struct {
	Index  int    `json:"index"`
	Size   int64  `json:"size"`
	Sha256 string `json:"sha256"`
}

// fileMeta хранится на диске и агрегирует информацию обо всём файле.
type fileMeta struct {
	FileID     string           `json:"file_id"`
	TotalParts int              `json:"total_parts"`
	Parts      map[int]partMeta `json:"parts"`
}

// writeMeta обновляет метаданные файла на диске.
func writeMeta(path string, fileID string, idx int, size int64, sha string, total int) error {
	fm := fileMeta{
		FileID:     fileID,
		TotalParts: total,
		Parts:      map[int]partMeta{},
	}

	// Если meta.json существует, загружаем текущую структуру перед обновлением.
	if b, err := os.ReadFile(path); err == nil {
		err = json.Unmarshal(b, &fm)
		if err != nil {
			return err
		}
	}

	// Обновляем запись для конкретной части, перезаписывая размер/хеш при повторном PUT.
	fm.Parts[idx] = partMeta{
		Index:  idx,
		Size:   size,
		Sha256: sha,
	}

	b, err := json.MarshalIndent(fm, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, b, 0o644)
}

// readMeta читает метаданные файла с диска.
func readMeta(path string) (*fileMeta, error) {
	// Храним JSON в удобочитаемом виде — ReadFile достаточно, т.к. размер meta.json мал.
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var fm fileMeta
	if err := json.Unmarshal(b, &fm); err != nil {
		return nil, err
	}

	return &fm, nil
}
