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


Файлы большего размера
https://ash-speed.hetzner.com/ - скачиваем 2 типа файлы: 1ГБ, 10ГБ

```bash
UPLOAD_RESP=$(curl -s -X POST \
  -H 'Content-Type: application/octet-stream' \
  --upload-file 1GB.bin \
  http://localhost:8080/files)

FILE_ID=$(echo "$UPLOAD_RESP" | jq -r '.file_id')
echo "FILE_ID=$FILE_ID"

curl -s -o restored.bin "http://localhost:8080/files/${FILE_ID}"
diff 1GB.bin restored.bin && echo "OK"
```

```bash
UPLOAD_RESP=$(curl -s -X POST \                                 
  -H 'Content-Type: application/octet-stream' \
  --upload-file 10GB.bin \
  http://localhost:8080/files)

FILE_ID=$(echo "$UPLOAD_RESP" | jq -r '.file_id')
echo "FILE_ID=$FILE_ID"

curl -s -o restored.bin "http://localhost:8080/files/${FILE_ID}"
diff 10GB.bin restored.bin && echo "OK"
```


## Тесты

```bash
go test ./...
```

## Go version

- build image: `golang:1.24`

## Конфиг

- `CONFIG_PATH` (по умолчанию `./config.yaml`)
- ENV override: `LISTEN_ADDR`, `META_DSN`, `STORAGES`
- Для сервиса метаданных используется только Postgres (`meta_dsn`). Для тестов/локальной отладки доступна спец-строка `memory://<name>`, которая хранит данные в памяти.

## Миграции

- SQL-миграции для Postgres лежат в `internal/repo/migrations` в формате `goose`.
- Для их применения добавлена команда `cmd/migrate`:
  ```bash
  go run ./cmd/migrate
  ```
  Команда читает тот же конфиг/ENV, что и REST, и применяет миграции из embedded-ресурсов.
- В Docker Compose миграции запускаются отдельным сервисом `migrator` до старта REST.

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
