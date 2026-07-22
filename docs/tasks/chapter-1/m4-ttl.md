# Веха M4. TTL ссылок и работа со временем

## Связанные темы
- 1.7.1 Пакет time: time.Time, Duration, форматирование и часовые пояса
- 1.7.2 Таймеры, тикеры и монотонные часы: подводные камни

## Результат вехи
Каждая короткая ссылка получает срок действия (TTL по умолчанию 24 часа). `Resolve` лениво проверяет TTL и возвращает `ErrExpired` для истёкших ссылок (пока заглушка `errors.New`; полноценная sentinel-ошибка — веха M5). Приложение начинает учитывать время жизни ссылок.

## Что собираем (шаг к результату)

### 1. Поле TTL в `Link` (в `internal/store/link.go`)
- Добавить поле `TTL time.Duration` в финальную структуру `Link` из M2:
  ```go
  type Link struct {
      Code    string
      LongURL string `json:"long_url"`
      TTL     time.Duration
      Hits    int64
      Audit
  }
  ```
- Метод:
  ```go
  // IsExpired возвращает true, если с момента CreatedAt прошло больше TTL.
  func (l *Link) IsExpired(at time.Time) bool {
      return at.Sub(l.CreatedAt) > l.TTL
  }
  ```
- `CreatedAt` доступен напрямую через встроенный `Audit` (промоутинг), поэтому `l.CreatedAt` работает.
- Константа:
  ```go
  // DefaultTTL — срок жизни ссылки по умолчанию.
  const DefaultTTL = 24 * time.Hour
  ```

### 2. TTL в `Shorten` (в `internal/shortener/shortener.go`)
- В `Shorten` при создании `Link` (из M2) устанавливать `TTL: DefaultTTL` (поле `Audit.CreatedAt` уже задаётся в M2 как `time.Now()`).
- Использовать `time.Now()` — возвращает `time.Time` с монотонным компонентом (важно для корректного измерения интервалов).
- Настройка TTL через опцию появится в Главе 8 (Functional Options).

### 3. Проверка TTL в `Resolve` (в `internal/store/memory.go`)
- В `MemoryStorage.Resolve`:
  ```go
  link, ok := m.links[code]
  if !ok {
      return nil, ErrNotFound
  }
  if link.IsExpired(time.Now()) {
      return nil, ErrExpired
  }
  return link, nil
  ```
- `ErrExpired` — sentinel (полная реализация ошибок — веха M5; здесь — `errors.New`).

## Отработка тем (практика и демо)

### Демонстрация `time.Timer` и `time.Ticker` (в `internal/store/link.go`)
- Завести демонстрационную функцию `demoTimers()`:
  - Создать `timer := time.NewTimer(50 * time.Millisecond)`, ждать `<-timer.C` (без конкурентности — в `main` или smoke-функции).
  - Создать `ticker := time.NewTicker(20 * time.Millisecond)`, сделать 3 тика, остановить через `ticker.Stop()`.
  - Логировать моменты срабатывания через `time.Now().Format(time.RFC3339Nano)`.
- Комментарий: объяснить разницу между `time.After`, `time.NewTimer`, `time.NewTicker` и когда что применять.

### Форматирование и часовые пояса (в `internal/store/link.go`)
- В `Link.String()` (реализован в M2) используется `l.CreatedAt.Format(time.RFC3339)` — убедиться, что поле `CreatedAt` корректно форматируется.
- Демонстрация `time.LoadLocation("Europe/Moscow")` и преобразование `t.In(loc)` — показать, что монотонный компонент сохраняется при конвертации зон.
- Комментарий: `time.Now()` включает монотонные часы; `t.Add(...)` и `t.Sub(...)` используют их, что защищает от скачков системных часов.

## Подсказки / подводные камни
- Использовать `time.Duration` (не `int` секунд) для всех интервалов.
- `time.Now()` — для измерения «сейчас»; не хранить «срок» как абсолютный `time.Time` без необходимости (TTL как `Duration` гибче).
- `time.Now()` содержит монотонный компонент; `time.Date(...)` и `time.Parse(...)` — нет. Сравнение `t1.After(t2)` между «обычной» и «монотонной» `time.Time` игнорирует монотонную часть.
- `time.AfterFunc` не запускается при `GOMAXPROCS=0` (теоретически); в реальности используйте `Timer`/`Ticker` корректно.
- Не забывайте `ticker.Stop()` / `timer.Stop()` — утечка ресурсов.
- `time.Sleep` не прерывается контекстом; для отменяемого ожидания — `Timer` + `select` (полноценно — Глава 2).
- `time.Duration` — это `int64` наносекунд; `24 * time.Hour` корректно, а `24 * 60 * 60 * 1e9` — нетипично и хрупко.
- Ленивая проверка TTL проще активной (через `Timer` на каждую ссылку); активная очистка появится в Главе 2 (reaper).

## Критерии готовности
- [ ] `Link` имеет поле `TTL` (в финальной модели из M2).
- [ ] `IsExpired(at time.Time) bool` реализован (pointer-receiver, согласован с M2).
- [ ] `Shorten` устанавливает `TTL: DefaultTTL` при создании `Link`.
- [ ] `Resolve` проверяет TTL и возвращает `ErrExpired` (заглушка).
- [ ] `DefaultTTL = 24 * time.Hour` объявлен.
- [ ] `demoTimers` демонстрирует `Timer` и `Ticker`.
- [ ] Форматирование через `time.RFC3339` используется в `String()`.
- [ ] Демонстрация `time.LoadLocation` и `t.In(loc)` присутствует.
- [ ] `go build ./...`, `go vet ./...`, `gofmt -l .` проходят чисто.

## Оценка времени
~6 часов.