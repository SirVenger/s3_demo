# storage_lite

## Компоненты
- REST-сервис (`cmd/rest`) принимает клиентские запросы и оркестрирует загрузку/чтение файлов.
- Узлы хранения (`cmd/storage`) принимают части файлов и складывают их на диск.

## Быстрый старт (Docker Compose)
1. Скопируйте пример конфига и при необходимости поправьте список стораджей:
   ```bash
   cp docker/rest-config.example.yaml docker/rest-config.yaml
   ```
2. Поднимите весь стенд:
   ```bash
   docker compose -f docker/docker-compose.yml up --build
   ```
   Если у вас установлена старая утилита `docker-compose`, выполните тот же запуск через `docker-compose -f docker/docker-compose.yml up --build`.
   После запуска REST доступен на `http://localhost:8080`, стораджи — на `http://localhost:808{1..6}`.

## Пример использования REST API
### Цельная загрузка/скачивание
```bash
printf 'demo data' > sample.bin

# загрузка файла
UPLOAD_RESP=$(curl -s -X POST \
  -H 'Content-Type: application/octet-stream' \
  --data-binary @sample.bin \
  http://localhost:8080/files)
FILE_ID=$(echo "$UPLOAD_RESP" | jq -r '.file_id')

# скачивание и проверка
curl -s -o restored.bin "http://localhost:8080/files/${FILE_ID}"
diff sample.bin restored.bin && echo "OK"
```

## Тесты
```bash
go test ./...
```

## Go version
- build image: `golang:1.24`

## Конфиг
- `CONFIG_PATH` (по умолчанию `./config.yaml`)
- ENV override: `LISTEN_ADDR`, `META_PATH`, `STORAGES`

## Endpoints
- `POST /files` — загрузка цельного файла (разрезаем на 6 частей)
- `GET /files/{id}` — чтение файла
- Админ: `GET /admin/config`, `GET /health`

## Storage API
- `PUT /parts/{fileID}/{idx}` (+ headers: `Content-Length`, `X-Checksum-Sha256` (optional), `X-Total-Parts`)
- `HEAD /parts/{fileID}/{idx}` → `X-Size`, `X-Checksum-Sha256`
- `GET /parts/{fileID}/{idx}`
- `POST /admin/gc` — ручной GC

## GC
На сторадж-нодах удаляются каталоги незавершённых загрузок, старше TTL.
Настройки: `GC_TTL_HOURS` (24), `GC_INTERVAL_MIN` (30).
