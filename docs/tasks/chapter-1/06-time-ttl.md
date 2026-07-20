# Задача 06. Работа со временем: TTL ссылок

## Связанные темы
- 1.7.1 Пакет time: time.Time, Duration, форматирование и часовые пояса
- 1.7.2 Таймеры, тикеры и монотонные часы: подводные камни

## Цель
Добавить поддержку TTL (времени жизни) коротких ссылок: каждая ссылка имеет срок действия, по истечении которого `Resolve` возвращает `ErrExpired`. Реализовать ленивую проверку TTL при `Resolve` и демонстрацию `time.Timer`.

## Что нужно сделать

### 1. Поля `Link`, связанные со временем (`internal/store/link.go`)
```go
type Link struct {
    Code      string
    LongURL   string
    CreatedAt time.Time
    TTL       time.Duration
}
```
- Метод:
  ```go
  // IsExpired возвращает true, если с момента CreatedAt прошло больше TTL.
  func (l *Link) IsExpired(at time.Time) bool {
      return at.Sub(l.CreatedAt) > l.TTL
  }
  ```
- Константа:
  ```go
  // DefaultTTL — срок жизни ссылки по умолчанию.
  const DefaultTTL = 24 * time.Hour
  ```

### 2. TTL в `Shorten`
- В `Shorten` при создании `Link` устанавливать `CreatedAt: time.Now()` и `TTL: DefaultTTL` (или значение, переданное через опцию — задача 07/Functional Options появится в Главе 8).
- Использовать `time.Now()` — возвращает `time.Time` с монотонным компонентом (важно для корректного измерения интервалов).

### 3. Проверка TTL в `Resolve`
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
- `ErrExpired` — sentinel (полная реализация ошибок — задача 09; здесь — `errors.New`).

### 4. Демонстрация `time.Timer` и `time.Ticker`
- Завести демонстрационную функцию `demoTimers()`:
  - Создать `timer := time.NewTimer(50 * time.Millisecond)`, ждать `<-timer.C` (без конкурентности — в `main` или smoke-функции).
  - Создать `ticker := time.NewTicker(20 * time.Millisecond)`, сделать 3 тика, остановить через `ticker.Stop()`.
  - Логировать моменты срабатывания через `time.Now().Format(time.RFC3339Nano)`.
- Комментарий: объяснить разницу между `time.After`, `time.NewTimer`, `time.NewTicker` и когда что применять.

### 5. Форматирование и часовые пояса
- В `Link.String()` (полная реализация — задача 07) использовать `l.CreatedAt.Format(time.RFC3339)`.
- Демонстрация `time.LoadLocation("Europe/Moscow")` и преобразование `t.In(loc)` — показать, что монотонный компонент сохраняется при конвертации зон.
- Комментарий: `time.Now()` включает монотонные часы; `t.Add(...)` и `t.Sub(...)` используют их, что защищает от скачков системных часов.

## Требования к коду
- Использовать `time.Duration` (не `int` секунд) для всех интервалов.
- `time.Now()` — для измерения «сейчас»; не хранить «срок» как абсолютный `time.Time` без необходимости (TTL как `Duration` гибче).
- Комментарии на русском объясняют монотонные часы и подводные камни таймеров.

## Критерии готовности
- [ ] `Link` имеет поля `CreatedAt` и `TTL`.
- [ ] `IsExpired(at time.Time) bool` реализован.
- [ ] `Resolve` проверяет TTL и возвращает `ErrExpired`.
- [ ] `demoTimers` демонстрирует `Timer` и `Ticker`.
- [ ] Форматирование через `time.RFC3339` используется в `String()`.
- [ ] `go build ./...`, `go vet ./...`, `gofmt -l .` проходят чисто.

## Оценка времени
~6 часов.

## Подсказки / подводные камни
- `time.Now()` содержит монотонный компонент; `time.Date(...)` и `time.Parse(...)` — нет. Сравнение `t1.After(t2)` между «обычной» и «монотонной» `time.Time` игнорирует монотонную часть.
- `time.AfterFunc` не запускается при `GOMAXPROCS=0` (теоретически); в реальности используйте `Timer`/`Ticker` корректно.
- Не забывайте `ticker.Stop()` / `timer.Stop()` — утечка ресурсов.
- `time.Sleep` не прерывается контекстом; для отменяемого ожидания — `Timer` + `select` (полноценно — Глава 2).
- `time.Duration` — это `int64` наносекунд; `24 * time.Hour` корректно, а `24 * 60 * 60 * 1e9` — нетипично и хрупко.
- Ленивая проверка TTL проще активной (через `Timer` на каждую ссылку); активная очистка появится в Главе 2 (reaper).
