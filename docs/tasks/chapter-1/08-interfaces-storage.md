# Задача 08. Интерфейсы: абстракция хранилища

## Связанные темы
- 1.9.1 Интерфейсы: концепция неявной реализации
- 1.9.2 Лучшие практики проектирования интерфейсов (малые интерфейсы, accept interfaces return structs)
- 1.9.3 Пустой интерфейс, any и type assertion / type switch
- 1.9.4 Тонкости выбора ресивера (value vs pointer) при реализации интерфейсов
- 1.9.5 Принципы ООП применительно к Go (SOLID)
- 1.9.6 Особенности работы со срезами и картами интерфейсов
- 1.9.7 Nil-интерфейс vs интерфейс с nil-значением

## Цель
Выделить интерфейс `Storage` в `internal/store`, чтобы `internal/shortener` зависел от интерфейса, а не от конкретного `MemoryStorage`. Применить принцип `accept interfaces, return structs` и продемонстрировать тонкости интерфейсов.

## Что нужно сделать

### 1. Интерфейс `Storage` (`internal/store/store.go`)
```go
// Storage описывает хранилище маппингов code → Link.
// Малый интерфейс: только то, что нужно пакету shortener.
type Storage interface {
    Save(link *Link) error
    Resolve(code string) (*Link, error)
    Delete(code string) error
    All() iter.Seq2[*Link, error]
}
```
- `MemoryStorage` реализует интерфейс **неявно** — без `implements` (тема 1.9.1).
- Добавить compile-time проверку:
  ```go
  var _ Storage = (*MemoryStorage)(nil)
  ```
  Это ловит несоответствие сигнатур на этапе компиляции.

### 2. `shortener` зависит от интерфейса
- В `internal/shortener`:
  ```go
  type Shortener struct {
      store store.Storage   // интерфейс, не конкретный тип
  }
  // NewShortener принимает интерфейс Storage, возвращает конкретный *Shortener.
  func NewShortener(s store.Storage) *Shortener
  ```
- Конструктор `NewShortener` принимает интерфейс (accept interfaces), возвращает структуру (return structs).
- `cmd/shorten/main.go` собирает зависимости:
  ```go
  mem := store.NewMemoryStorage()
  s := shortener.NewShortener(mem)
  ```

### 3. Ресивер и реализация интерфейса (тема 1.9.4)
- Если методы `Storage` объявлены с pointer-receiver (`func (m *MemoryStorage) Save`), то интерфейс реализует только `*MemoryStorage`, а не `MemoryStorage`.
- Демонстрация: `var s Storage = mem` (где `mem *MemoryStorage`) работает; `var s Storage = *mem` — нет.
- Комментарий: объяснить, почему `NewMemoryStorage` возвращает указатель.

### 4. `any`, type assertion, type switch (тема 1.9.3)
- Завести демонстрационную функцию `describe(v any) string`:
  ```go
  switch v := v.(type) {
  case *Link:
      return "link: " + v.Code
  case string:
      return "string: " + v
  case nil:
      return "nil"
  default:
      return fmt.Sprintf("unknown: %T", v)
  }
  ```
- Использовать `any` (алиас `interface{}` с Go 1.18+) в сигнатуре.
- Демонстрация безопасной type assertion: `if l, ok := v.(*Link); ok { ... }`.

### 5. Nil-интерфейс vs интерфейс с nil-значением (тема 1.9.7)
- Демонстрационная функция `demoNilInterface()`:
  ```go
  var s Storage        // nil-интерфейс
  fmt.Println(s == nil) // true
  var mem *MemoryStorage = nil
  var s2 Storage = mem
  fmt.Println(s2 == nil) // false! интерфейс не nil, хотя значение nil
  ```
- Комментарий: интерфейс — это пара `(type, value)`; он равен `nil` только если обе компоненты `nil`. Вызов метода на `s2` паникует, если метод не проверяет nil-приёмник.

### 6. Срезы и карты интерфейсов (тема 1.9.6)
- Демонстрация: `[]Storage` со ссылкой на несколько реализаций (пока одна `MemoryStorage`, плюс можно добавить `NoopStorage` для теста).
- Комментарий: хранение интерфейсов в срезах/картах боксит значения — стоимость аллокации; не используйте интерфейсы без необходимости.

### 7. SOLID (тема 1.9.5)
- Краткий комментарий в `store.go`:
  - **S** (SRP): `Storage` отвечает только за хранение.
  - **O** (OCP): новая реализация (например, `PostgresStorage` в Главе 3) добавляется без изменения `shortener`.
  - **L** (LSP): любая реализация `Storage` взаимозаменяема.
  - **I** (ISP): интерфейс малый — только методы, нужные потребителю.
  - **D** (DIP): `shortener` зависит от абстракции `Storage`, не от конкретики.

## Требования к коду
- Интерфейсы объявляются в пакете-потребителе или в `store` — малые, по 3–4 метода.
- `var _ Storage = (*MemoryStorage)(nil)` — compile-time гарантия реализации.
- `NewShortener` принимает интерфейс; `NewMemoryStorage` возвращает конкретный тип.
- Комментарии на русском объясняют неявную реализацию, nil-интерфейс, SOLID.

## Критерии готовности
- [ ] Интерфейс `Storage` объявлен.
- [ ] `MemoryStorage` реализует `Storage` (compile-time проверка присутствует).
- [ ] `Shortener` зависит от `store.Storage`, не от `*MemoryStorage`.
- [ ] `main` собирает зависимости через конструкторы.
- [ ] `describe` демонстрирует type switch и type assertion.
- [ ] `demoNilInterface` демонстрирует nil-интерфейс vs nil-значение.
- [ ] Комментарии по SOLID добавлены.
- [ ] `go build ./...`, `go vet ./...`, `gofmt -l .` проходят чисто.

## Оценка времени
~7 часов.

## Подсказки / подводные камни
- Интерфейс с nil-значением — классический баг: проверка `if s != nil` проходит, но вызов метода паникует. Возвращайте `nil`-интерфейс, а не типизированный nil.
- Не объявляйте интерфейс «на всякий случай»; объявляйте там, где есть несколько реализаций или где нужен mock для тестов.
- Методы с value-receiver реализуют интерфейс и значением, и указателем; с pointer-receiver — только указателем.
- Боксинг интерфейсов — аллокация; в hot path избегайте `[]interface{}`.
- `any` и `interface{}` — синонимы с Go 1.18; предпочитайте `any` для читаемости.
- Малые интерфейсы (`io.Reader`, `io.Writer`) — идиома Go; не плодите «жирные» интерфейсы.
