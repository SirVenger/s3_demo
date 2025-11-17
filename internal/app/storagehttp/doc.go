// Package storagehttp реализует Storage API — HTTP-интерфейс стораджа, принимающего и
// выдающего части файлов поверх локального диска. Основные эндпоинты:
//   - PUT /parts/{fileID}/{idx} — принимает часть, проверяет размер/хеш и сохраняет вместе с meta.json.
//   - GET /parts/{fileID}/{idx} — отдаёт сохранённую часть как application/octet-stream.
//   - HEAD /parts/{fileID}/{idx} — возвращает размер и SHA-256 через служебные заголовки.
//   - POST /admin/gc — инициирует сбор незавершённых загрузок (ручной GC).
//   - GET /health — отдаёт агрегированные метрики по каталогу данных для health-check'ов.
package storagehttp
