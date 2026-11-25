# TripGo — Trip Service

HTTP API поездок и PostgreSQL из лабораторной работы 1; структура кода переносится на слои лабораторной работы 2. Сервис создаёт поездку, возвращает её по ID и завершает. Контракт API находится в [`contracts/openapi/trip-service.openapi.yaml`](contracts/openapi/trip-service.openapi.yaml).

## Требования

- Go 1.27 (версия указана в `go.mod`);
- Docker и `tripgoctl` для локального PostgreSQL;
- `make`.

Инструменты `goose` и `oapi-codegen` закреплены в `go.mod` как Go tools. Отдельно устанавливать их не нужно.

## Запуск

Из корня репозитория:

```bash
tripgoctl cluster start
tripgoctl environment start
tripgoctl connect
make migrate
make run
```

`tripgoctl connect` создаёт `.env` с адресом локальной БД. Этот файл и `.tripgo/` не коммитятся. `Makefile` сначала загружает `.env.example`, затем значения из `.env`; так собственные параметры из примера доступны вместе с адресами, выданными `tripgoctl`. Для `run` и миграций `.env` обязателен. Порт HTTP по умолчанию — `8080`.

Проверка запущенного сервиса:

```bash
curl -i http://localhost:8080/health
curl -i http://localhost:8080/ready
```

Для ручной проверки поездок импортируйте в Postman [коллекцию](postman/tripgo-lab1.postman_collection.json) и [окружение](postman/tripgo-local.postman_environment.json). Выберите `TripGo Local` и выполняйте папку «Основной сценарий» по порядку: после создания коллекция сохранит ID в `trip_id`.

## Команды

| Команда | Назначение |
|---|---|
| `make run` | Запустить сервис |
| `make migrate` | Применить все новые миграции |
| `make migrate-status` | Показать состояние миграций |
| `make migrate-down` | Откатить одну миграцию |
| `make generate` | Пересоздать код из OpenAPI |
| `make generate-mocks` | Пересоздать моки через `mockgen` |
| `make test` | Unit- и HTTP-тесты с `-race`, показать покрытие домена и usecase |
| `make test-integration` | Применить миграции и дважды выполнить тесты репозитория на PostgreSQL |
| `make lint` | Проверить код линтерами и форматтерами |

`make migrate-down` меняет схему и может удалить данные. Для проверки полного отката двух миграций используйте отдельную тестовую БД.

## Переменные окружения ЛР1

Имена и локальные значения по умолчанию находятся в [`.env.example`](.env.example). Обязательные параметры приложения проверяются при запуске сервиса; `GOOSE_DRIVER` используется только командами миграций. `DATABASE_URL` для текущего окружения берётся из `.env`, созданного `tripgoctl`.

| Переменная | Назначение |
|---|---|
| `HTTP_ADDR` | Адрес HTTP-сервера |
| `HTTP_READ_TIMEOUT`, `HTTP_READ_HEADER_TIMEOUT` | Ограничения времени чтения запроса и заголовков |
| `HTTP_WRITE_TIMEOUT`, `HTTP_IDLE_TIMEOUT` | Ограничения времени ответа и простоя соединения |
| `SHUTDOWN_TIMEOUT` | Максимальное ожидание активных запросов при остановке |
| `DATABASE_URL` | Строка подключения к PostgreSQL |
| `DATABASE_MAX_CONNS`, `DATABASE_MIN_CONNS` | Размеры пула соединений |
| `DATABASE_MAX_CONN_LIFETIME` | Максимальное время жизни соединения |
| `DATABASE_CONNECT_TIMEOUT` | Таймаут установки соединения |
| `DATABASE_QUERY_TIMEOUT` | Таймаут операций с БД |
| `DATABASE_PING_TIMEOUT` | Таймаут проверки доступности БД |
| `GOOSE_DRIVER` | Драйвер миграций (`postgres`) |
| `LOG_LEVEL` | Уровень JSON-логов: `debug`, `info`, `warn` или `error` |

Переменные `OTEL_*` из `.env.example` задают имя сервиса, адрес коллектора, протокол `http/protobuf` и сэмплирование трейсов. `tripgoctl connect` записывает актуальный адрес коллектора в `.env`. Остальные переменные следующих лабораторных работ текущий сервис пока не использует.

## HTTP API

| Метод и путь | Успех | Ошибки |
|---|---|---|
| `POST /api/v1/trips` | `201`, поездка и `Location` | `400 invalid_request`, `409 driver_busy`, `500 internal_error` |
| `GET /api/v1/trips/{tripId}` | `200`, поездка | `400 invalid_request`, `404 trip_not_found`, `500 internal_error` |
| `POST /api/v1/trips/{tripId}/finish` | `200`, завершённая поездка | `400 invalid_request`, `404 trip_not_found`, `409 trip_completed`, `500 internal_error` |
| `GET /health` | `200 {"status":"ok"}` | — |
| `GET /ready` | `200 {"status":"ok"}` | `503 {"status":"unavailable"}` |

Ошибки поездок возвращаются с `Content-Type: application/problem+json`. Внутренние ошибки записываются в лог, но текст ошибки БД не включается в ответ клиенту. `/health` не обращается к БД; `/ready` проверяет её через `Ping`.

## Принятые решения

### Архитектура

- `internal/http/trip/{create,get,finish}` разбирает запросы и вызывает соответствующий usecase. `internal/http/router.go` связывает эти обработчики со сгенерированным OpenAPI-интерфейсом, а `internal/http/domainerror` переводит доменные ошибки в `Problem`.
- `internal/usecase/trip/{create,get,finish}` содержит сценарии работы. Каждый пакет объявляет в своём `contract.go` только нужные ему интерфейсы репозиториев и менеджера транзакций.
- `internal/domain/trip` хранит правила перехода статуса и доменные ошибки; `internal/model` содержит структуры данных, передаваемые между слоями. Домен не импортирует HTTP или PostgreSQL.
- `internal/repository/trip` реализует SQL-запросы и объявляет минимальный интерфейс исполнителя в своём `contract.go`. `internal/infra/postgres` создаёт пул, а `internal/infra/postgres/transaction` управляет транзакциями и хранит текущую транзакцию в контексте.
- `internal/container` создаёт конкретные реализации и соединяет их. `cmd/trip-service/main.go` читает конфигурацию, запускает сервер и завершает его по сигналу.
- `internal/infra/telemetry` настраивает OpenTelemetry SDK и JSON-логгер. HTTP middleware идут в порядке `Recover → Tracing → Logging → Metrics`; `otelpgx` создаёт спаны запросов PostgreSQL, а измеритель рядом с пулом записывает длительность с именами операций и таблиц. Usecase сообщает об успешном создании и завершении поездки через узкий интерфейс.

Доменный `Trip.Finish` проверяет переход `active → completed`; условный `UPDATE` в репозитории повторно проверяет статус в PostgreSQL, чтобы два конкурентных запроса не завершили одну поездку дважды. Часы и генератор UUID передаются в usecase как зависимости, чтобы позднее проверять их детерминированно.

**Транзакции.** `TxManager.Do` начинает транзакцию, помещает её в `context.Context` и передаёт контекст функции. Репозитории берут исполнитель из контекста: при наличии транзакции используют её, иначе пул. Бизнес-код не зависит от `pgx`. Успешный вызов завершается `COMMIT`, ошибка или паника — `ROLLBACK`; вложенный `Do` переиспользует текущую транзакцию. Создание поездки записывает `trips` и `trip_status_history` внутри одного `Do`.

**Изоляция.** Выбран `Read Committed`. Для этих операций не нужна более строгая изоляция: запрет двух активных поездок закреплён уникальным индексом БД, а завершение выполняется условным `UPDATE`. После безрезультатного `UPDATE` повторное чтение различает отсутствующую и уже завершённую поездку.

**Один активный заказ на водителя.** Частичный уникальный индекс `trips_one_active_per_driver_idx` действует только для строк со статусом `active`. PostgreSQL обеспечивает правило при параллельных запросах; нарушение индекса с кодом `23505` преобразуется в доменную ошибку, а HTTP-слой возвращает `409 driver_busy`.

**Остановка.** По `SIGINT` или `SIGTERM` сервер прекращает принимать новые запросы и ждёт активные не дольше `SHUTDOWN_TIMEOUT`. Затем закрывается пул PostgreSQL. Ошибка запуска или остановки приводит к ненулевому коду выхода.

## Проверка

```bash
go build ./...
make lint
make test
make migrate-status
make generate
```

Для проверки бизнес-правил: повторите создание с тем же `driver_id` до завершения первой поездки — ожидается `409 driver_busy`; завершите поездку дважды — `200`, затем `409 trip_completed`. При двадцати одновременных созданиях для одного водителя должна появиться одна поездка; при двух одновременных завершениях — один `200` и один `409`.

Для проверки телеметрии выполните HTTP-запрос и откройте Jaeger и Grafana по адресам из `tripgoctl connect`. В Jaeger у HTTP-спана должны быть вложенные спаны PostgreSQL; `trace_id` из JSON-лога должен находить тот же трейс. В Prometheus/Grafana доступны `http_server_request_duration_seconds`, `db_client_operation_duration_seconds`, `tripgo_trips_created_total`, `tripgo_trips_completed_total` и `tripgo_trips_active`.

### Тесты ЛР2

`make test` не требует запущенной БД. Моки зависимостей usecase и HTTP-хендлеров генерируются из их `contract.go` командой `make generate-mocks`; сгенерированные `mocks_test.go` хранятся в репозитории. Покрытие `internal/domain` и `internal/usecase` печатается последней строкой.

Для `make test-integration` сначала выполните `tripgoctl environment start` и `tripgoctl connect`. Команда применит миграции и запустит тесты с билд-тегом `integration` на реальной PostgreSQL. Каждый тест использует случайные UUID и удаляет свои записи после завершения; сценарий отката проверяет отсутствие строк в обеих таблицах.

### Что смотреть в Grafana

Откройте адрес Grafana из `tripgoctl connect` → **Explore** → источник **Prometheus** → режим **Code**, выберите диапазон **Last 1 hour** и выполните запрос. После новых HTTP-запросов полезны:

```promql
sum by (http_route, http_response_status_code) (http_server_request_duration_seconds_count)
```

```promql
sum by (db_collection_name, db_operation_name) (db_client_operation_duration_seconds_count)
```

```promql
tripgo_trips_created_total
```

Также проверьте `tripgo_trips_completed_total` и `tripgo_trips_active`. Для PR сделайте скриншот Explore с HTTP-метрикой и счётчиком поездок; в Jaeger отдельно покажите HTTP-спан с вложенным запросом PostgreSQL.

Идемпотентность по `Idempotency-Key` и Dockerfile относятся к заданиям со звёздочкой и здесь не реализованы. Координаты, gRPC и события относятся к следующим лабораторным работам.
