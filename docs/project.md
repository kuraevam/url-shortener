# Эволюционный проект для закрепления Go

## Главная рекомендация: «URL Shortener + Analytics» (сокращатель ссылок с аналитикой)

Это классическая система, которая **органично растёт вместе с вашим знанием Go** — от консольной утилиты до распределённого микросервиса с observability. Каждый раздел курса добавляет новый слой фич и инфраструктуры, а не «приклеивается сбоку».

---

## Развитие проекта по главам

### Глава 1 — Введение в Go
**Что строим:** CLI-утилиту `shorten`, которая генерирует короткие коды и хранит маппинг `code → url` в памяти (в `map`).

| Тема | Применение в проекте |
|---|---|
| 1.1–1.3 | Структура пакета `cmd/shorten`, `go build`, `gofmt`, `go vet` |
| 1.2 | Парсинг аргументов, функции `Shorten(longURL)`, `Resolve(code)` |
| 1.4 | Срезы для истории переходов, `append` |
| 1.5 | Генерация кода через `strings.Builder`, валидация URL через `regexp` |
| 1.6 | `map[string]string` — хранилище маппингов |
| 1.7 | TTL ссылок через `time.Timer` |
| 1.8 | `struct Link { Code, LongURL, CreatedAt }`, методы `String()` |
| 1.9 | Интерфейс `Storage` (memory impl) — **accept interfaces, return structs** |
| 1.10 | Кастомные ошибки `ErrNotFound`, `ErrExpired`; `defer` для закрытия файла |
| 1.11 | Разбивка на пакеты `internal/store`, `internal/shortener`, `go.mod` |
| 1.12 | Дженерик-функция `Filter[T](items []T, pred func(T) bool)` |
| 1.13 | Итератор `All()` поверх хранилища через `range-over-func` |

<details>
<summary>Делеверабл</summary>

`shorten https://example.com` → печатает `https://s.io/AbC12x`. `shorten --resolve AbC12x` → оригинальный URL. Хранилище — `map` в памяти.
</details>

---

### Глава 2 — Параллельное программирование
**Что добавляем:** Фоновый воркер очистки истёкших ссылок + параллельная генерация батчей кодов.

- **2.1–2.2** — горутина-«уборщик», которая по `ticker` удаляет `expired` ссылки; `sync.WaitGroup` для graceful shutdown.
- **2.3** — канал `events chan Visit` для асинхронной записи переходов; `select` с `ctx.Done()`.
- **2.4** — `sync.RWMutex` на хранилище; `race detector` (`-race`).
- **2.5** — `atomic.Int64` для счётчика переходов; `sync.Pool` для буферов логов.
- **2.6** — `context.Context` пробрасывается во все методы; `WithTimeout` на операции.
- **2.7** — паттерн **pipeline**: `fetchURL → validate → generateCode → save`; **fan-out** для генерации N кодов параллельно.
- **2.8** — `errgroup.Group` для параллельной проверки доступности пачки URL.
- **2.9** — **worker pool** для обработки очереди переходов; **rate limiter** на генерацию кодов (`x/time/rate`).

<details>
<summary>Делеверабл</summary>

Фоновый `reaper` горутина, event-driven запись переходов через канал, worker pool из N воркеров, ограничение скорости через `rate.Limiter`.
</details>

---

### Глава 3 — Работа с данными
**Что добавляем:** Персистентное хранилище (PostgreSQL), кеш (Redis), аналитика переходов (MongoDB).

- **3.1** — Потоковая запись логов переходов через `io.Writer` / `bufio.Writer`.
- **3.2** — Флаги `--port`, `--dsn` через `flag` и `cobra`.
- **3.3** — Чтение/запись файла истории; `embed` для шаблона главной страницы.
- **3.4** — JSON-сериализация `Link`, REST-like формат данных; `encoding/json/v2` эксперимент.
- **3.5** — PostgreSQL через `pgx/v5` + `sqlc` для типобезопасных запросов; транзакции при создании ссылки + записи лога.
- **3.6** — Миграции `goose`: `links`, `visits` таблицы.
- **3.7** — Redis для кеша `code → url` (cache-aside), TTL; **Redlock** для распределённой блокировки при генерации уникального кода.
- **3.8** — MongoDB для аналитики: события переходов (IP, UA, geo, timestamp).
- **3.9** — Kafka для событий `link.created`, `link.visited` — консьюмеры пишут в ClickHouse/аналитику.

<details>
<summary>Делеверабл</summary>

Хранилище переключаемое: `MemoryStorage | PostgresStorage` (один интерфейс). Redis-кеш с инвалидацией по TTL. Kafka-продюсер событий.
</details>

---

### Глава 4 — Веб-разработка
**Что добавляем:** HTTP-сервер с REST API.

- **4.1** — `net/http` сервер: `POST /shorten`, `GET /:code` (redirect), `GET /:code/stats`.
- **4.2** — Роутер `chi`, middleware: logging, recovery, request-id.
- **4.3** — Валидация URL через `validator/v10`, чтение JSON-body, заголовков.
- **4.4** — Простая HTML-страница со списком Top-100 ссылок (`html/template`).
- **4.5** — HTTP-клиент для проверки доступности URL перед сокращением (с retry и timeout).
- **4.6** — По желанию — переход на `chi` или `Gin`.
- **4.7** — WebSocket для real-time счётчика переходов на главной.
- **4.8** — OpenAPI-спецификация (`swag`), версионирование `/api/v1/`.

<details>
<summary>Делеверабл</summary>

REST API: `POST /api/v1/shorten`, `GET /:code` (302 redirect), `GET /api/v1/links/:code/stats`. WebSocket `/ws/stats` с real-time обновлением.
</details>

---

### Глава 5 — Тестирование и отладка
**Что добавляем:** Покрытие тестами + бенчмарки.

- **5.1** — Table-driven тесты для `Shorten()`, `Resolve()`, `Validate()`.
- **5.2** — Моки интерфейсов `Storage`, `Cache`; `httptest` для handlers.
- **5.3** — Интеграционные тесты с `testcontainers-go` (PostgreSQL, Redis, Kafka).
- **5.4** — Бенчмарки генерации кодов, throughput редиректов.
- **5.5** — `pprof` endpoints, нагрузочный профиль; найти bottleneck в `Shorten()`.

---

### Глава 6 — Основы микросервисов
**Что добавляем:** Разбиваем на 2 сервиса + gRPC.

- **6.1** — Чистая архитектура: `domain/`, `usecase/`, `adapter/http`, `adapter/storage`.
- **6.3** — gRPC-сервис `Shortener` (unary) + `Analytics` (server-streaming для потока переходов).
- **6.4** — Outbox-паттерн: событие `link.visited` пишется в БД в той же транзакции, затем воркер отправляет в Kafka.
- **6.5** — Service discovery через Consul/etcd; API Gateway на chi.

<details>
<summary>Архитектура</summary>

```
[Client] → [API Gateway :8080]
              ↓ HTTP
   [Shortener service gRPC:9000] → [Postgres + Redis]
              ↓ event (Kafka / outbox)
   [Analytics service gRPC:9001] → [MongoDB / ClickHouse]
```
</details>

---

### Глава 7 — Безопасность
**Что добавляем:** Аутентификацию и TLS.

- **7.1–7.2** — HTTPS с самоподписанным сертификатом; mTLS между сервисами.
- **7.3** — JWT-аутентификация: `POST /register`, `POST /login` → access + refresh токены.
- **7.4** — OAuth2 login через Google/GitHub для личного кабинета ссылок.
- **7.5** — bcrypt для паролей; CORS; SQL-injection-безопасные запросы (sqlc); `govulncheck` в CI.

---

### Глава 8 — Продвинутая разработка
**Что добавляем:** Observability, DI, конфиги, генерация кода.

- **8.1** — Кастомный парсер struct-тегов для логирования или валидации.
- **8.2** — `google/wire` для DI: генерация `wire_gen.go` со сборкой всех зависимостей.
- **8.3** — Конфиг через `viper`: `config.yaml` + env overrides + флаги.
- **8.4** — Оптимизация `Shorten()`: снижение аллокаций (пул `[]byte`), `GOMEMLIMIT` в контейнере.
- **8.6** — `slog` для structured logging; **OpenTelemetry** traces; Prometheus-метрики (`/metrics`).
- **8.7** — Functional Options для `NewServer(opts...)`; Builder для конфигов.
- **8.8** — `go generate` + `stringer` для типа `LinkStatus`.
- **8.9** — Multi-stage Dockerfile → `distroless` образ, ~20 MB.

---

### Глава 9 — DevOps
**Что добавляем:** CI/CD + Kubernetes.

- **9.1** — `.golangci.yml` с расширенным набором линтеров.
- **9.2** — GitHub Actions: lint → test → build → release через `GoReleaser`.
- **9.3** — Helm-чарт: `Deployment`, `Service`, `Ingress`, `ConfigMap`, `Secret`; HPA по CPU.
- **9.4** — Production-readiness checklist; canary-деплой через 2 Deployment.

---

### Глава 10 — Системный дизайн и эксплуатация
**Что добавляем:** Масштабирование и SRE-практики.

- **10.1** — Capacity planning: 10K RPS редиректов, расчёт ресурсов.
- **10.2** — Многоуровневый кеш (Redis + in-memory LRU); **rate limiter** на генерацию (token bucket per IP); **circuit breaker** на вызовы Analytics.
- **10.3** — Практикум: симуляция memory leak (забыли `defer rows.Close()`), профилирование через `pprof`; утечка горутин (незакрытый канал).
- **10.4** — SLO: p99 < 50ms для редиректа; нагрузочное тестирование через `k6` или `vegeta`.
- **10.5** — Design doc для фичи «custom aliases + custom domain».

---

## Краткая карта эволюции

```
Глава 1:  CLI-утилита в памяти
Глава 2:  + горутины, worker pool, каналы
Глава 3:  + Postgres + Redis + Kafka
Глава 4:  + HTTP REST API
Глава 5:  + тесты, бенчмарки, pprof
Глава 6:  → микросервисы + gRPC + outbox
Глава 7:  + JWT, OAuth2, mTLS
Глава 8:  + slog, OpenTelemetry, wire, Docker
Глава 9:  + CI/CD, Helm, HPA
Глава 10: + rate limiter, circuit breaker, SLO, load testing
```

---

## Альтернативные проекты (коротко)

| Проект | Почему подходит | Особенность |
|---|---|---|
| **Note-taking / Pastebin** | CRUD + поиск + теги → отличная база для аутентификации и поиска | Проще для Chapter 4–7, но слабее для системного дизайна |
| **Мониторинг-агент** (сбор метрик с хостов) | Идеален для горутин, каналов, gRPC streaming, observability | Сильнее в concurrency/observability, слабее в REST/DB |
| **Task runner / cron-as-a-service** | Хорошо ложится на горутины, scheduler, persistence | Меньше веб-фич, но интересно для system design |

---

**Моя рекомендация:** стартуйте с **URL Shortener** прямо сегодня. Создайте `go.mod`, напишите первую версию из 50 строк, и с каждой новой темой возвращайтесь к проекту — рефакторите и расширяйте. Это даст вам работающий код на GitHub, который к концу обучения станет полноценным портфолио.