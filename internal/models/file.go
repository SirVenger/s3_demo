package models

// Part описывает одну часть файла, лежащую в узле хранения.
type Part struct {
	Index   int    `json:"index"`
	Size    int64  `json:"size"`
	Sha256  string `json:"sha256"`
	Storage string `json:"storage"`
}

// File содержит агрегированные метаданные о всех частях файла.
type File struct {
	ID         string       `json:"file_id"`
	Name       string       `json:"file_name,omitempty"`
	Size       int64        `json:"size"`
	TotalParts int          `json:"total_parts"`
	Parts      map[int]Part `json:"parts"`
}

// Clone возвращает копию структуры, чтобы не делиться внутренними картами.
func (f File) Clone() File {
	out := File{
		ID:         f.ID,
		Name:       f.Name,
		Size:       f.Size,
		TotalParts: f.TotalParts,
		Parts:      map[int]Part{},
	}
	for idx, part := range f.Parts {
		out.Parts[idx] = part
	}
	return out
}
