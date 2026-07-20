# Задача 05. Карты (map): хранилище маппингов

## Связанные темы
- 1.6.1 Карты: создание, доступ, удаление элементов
- 1.6.2 Итерация по карте и порядок обхода
- 1.6.3 Внутреннее устройство map (Swiss Tables в Go 1.24+) и особенности производительности

## Цель
Реализовать `MemoryStorage` на основе `map[string]*Link` для хранения маппингов `code → Link`. Заменить временную `map[string]string` из задачи 02 на полноценное хранилище (пока без интерфейса — задача 08).

## Что нужно сделать

### 1. Поля `MemoryStorage` (`internal/store/memory.go`)
```go
type MemoryStorage struct {
    links   map[string]*Link   // code → Link
    history []Visit            // из задачи 03
}
```
- Конструктор:
  ```go
  // NewMemoryStorage создаёт пустое хранилище в памяти.
  func NewMemoryStorage() *MemoryStorage {
      return &MemoryStorage{links: make(map[string]*Link)}
  }
  ```

### 2. Методы работы с картой
```go
// Save сохраняет ссылку. Возвращает ошибку, если код уже занят.
func (m *MemoryStorage) Save(link *Link) error
// Resolve возвращает Link по коду или ошибку.
func (m *MemoryStorage) Resolve(code string) (*Link, error)
// Delete удаляет ссылку по коду. Idempotent.
func (m *MemoryStorage) Delete(code string) error
// Len возвращает количество хранимых ссылок.
func (m *MemoryStorage) Len() int
```
- `Save`: проверка `if _, ok := m.links[link.Code]; ok { return ErrDuplicate }`, затем `m.links[link.Code] = link`.
- `Resolve`: `link, ok := m.links[code]`; если `!ok` → `ErrNotFound` (кастомные ошибки — задача 09; здесь — заглушки `errors.New`).
- `Delete`: `delete(m.links, code)` — idempotent, без ошибки при отсутствии.
- `Len`: `return len(m.links)`.

### 3. Итерация по карте
- Реализовать метод `ForEach(fn func(code string, link *Link))`:
  ```go
  for code, link := range m.links {
      fn(code, link)
  }
  ```
- Демонстрация случайного порядка обхода: добавить комментарий и smoke-проверку, что два последовательных `ForEach` могут выдавать ключи в разном порядке.
- Использовать `range` с двумя переменными (`code, link`).

### 4. Проверка существования ключа
- Везде, где нужно различать «ключа нет» и «нулевое значение», использовать форму `v, ok := m[k]` (comma-ok).
- Не использовать `m[k] == nil` для проверки существования (особенно важно для `*Link`).

### 5. Интеграция с `Shorten`/`Resolve`
- В `internal/shortener` завести поле `store *store.MemoryStorage` (инъекция через конструктор `NewShortener(s *store.MemoryStorage)`).
- `Shorten` вызывает `GenerateCode` (задача 04), создаёт `Link` (задача 07 — пока минимальный `&Link{Code: ..., LongURL: ...}`), вызывает `store.Save`.
- `Resolve` вызывает `store.Resolve` и возвращает `LongURL`.
- При коллизии кода (редко) — повторная генерация (простой цикл с лимитом попыток).

## Требования к коду
- `make(map[string]*Link)` — обязательная инициализация перед использованием.
- Проверка существования ключа — только через comma-ok.
- Комментарии на русском объясняют случайный порядок обхода и почему нельзя полагаться на него.

## Критерии готовности
- [ ] `MemoryStorage` использует `map[string]*Link`.
- [ ] `Save`/`Resolve`/`Delete`/`Len` реализованы.
- [ ] `ForEach` демонстрирует итерацию и случайный порядок.
- [ ] Проверка существования — через comma-ok.
- [ ] `Shorten`/`Resolve` интегрированы с хранилищем.
- [ ] `go build ./...`, `go vet ./...`, `gofmt -l .` проходят чисто.

## Оценка времени
~5 часов.

## Подсказки / подводные камни
- `nil`-карта доступна на чтение (возвращает нулевое значение), но `nil`-карта при записи паникует — всегда `make`.
- Порядок обхода `map` не детерминирован; не полагайтесь на него в логике.
- В Go 1.24+ внутреннее устройство map заменено на Swiss Tables — производительность улучшилась, но семантика API не изменилась.
- Не берите адрес элемента map (`&m[k]`) — он может стать невалидным после переаллокации.
- `delete` на отсутствующем ключе — no-op, не паникует.
- Итерация с модификацией карты во время `range` — поведение не определено; избегайте.
