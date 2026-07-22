# Веха M5. Интерфейс хранилища и полноценная обработка ошибок

## Связанные темы
- 1.9.1 Интерфейсы: концепция неявной реализации
- 1.9.2 Лучшие практики проектирования интерфейсов (малые интерфейсы, accept interfaces return structs)
- 1.9.3 Пустой интерфейс, any и type assertion / type switch
- 1.9.4 Тонкости выбора ресивера (value vs pointer) при реализации интерфейсов
- 1.9.5 Принципы ООП применительно к Go (SOLID)
- 1.9.6 Особенности работы со срезами и картами интерфейсов
- 1.9.7 Nil-интерфейс vs интерфейс с nil-значением
- 1.10.1 Философия обработки ошибок в Go (errors as values)
- 1.10.2 Создание собственных типов ошибок (sentinel-ошибки и кастомные типы)
- 1.10.3 Оборачивание ошибок (%w) и функции errors.Is / errors.As / errors.Join
- 1.10.4 defer: семантика LIFO, момент вычисления аргументов и базовые применения
- 1.10.5 defer в цикле, модификация именованных возвращаемых значений и стоимость defer
- 1.10.6 Механика panic и recover: когда и как применять

## Результат вехи
**Приложение полностью работоспособно.** `Shortener` зависит от интерфейса `Storage` (а не от конкретного `MemoryStorage`); заглушки `errors.New` из M2–M4 заменены на sentinel-ошибки `ErrNotFound`/`ErrExpired`/`ErrDuplicate`/`ErrInvalidURL` и кастомный тип `LinkError` с `Unwrap`; обёртка через `%w`; `cmd/shorten/main.go` мапит ошибки через `errors.Is` в человекочитаемые сообщения (`error: not found` и т.п.) с ненулевым кодом возврата. После этой вехи smoke-цикл `shorten`/`resolve` полностью соответствует спеке (раздел 4). Веха M6 расширяет хранилище дженериками/итераторами, но не обязательна для базового цикла.

## Что собираем (шаг к результату)

### 1. Интерфейс `Storage` (в `internal/store/store.go`)
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
- `MemoryStorage` реализует интерфейс **неявно** — без `implements`.
- Добавить compile-time проверку:
  ```go
  var _ Storage = (*MemoryStorage)(nil)
  ```
  Это ловит несоответствие сигнатур на этапе компиляции.
- `All()` реализуется в вехе M6; до тогда достаточно, чтобы сигнатура присутствовала в интерфейсе. Чтобы `MemoryStorage` удовлетворял интерфейсу уже сейчас, в `internal/store/iter.go` можно добавить заглушку `All()`, возвращающую пустой итератор (полноценная реализация — M6).

### 2. `shortener` зависит от интерфейса (в `internal/shortener/shortener.go`)
- **Заменить** поле `store *store.MemoryStorage` (из M2) на интерфейс:
  ```go
  type Shortener struct {
      store store.Storage   // интерфейс, не конкретный тип
  }
  // NewShortener принимает интерфейс Storage, возвращает конкретный *Shortener.
  func NewShortener(s store.Storage) *Shortener
  ```
- Сигнатура меняется с `*store.MemoryStorage` (M2) на `store.Storage`; тело `Shortener.Shorten`/`Shortener.Resolve` остаётся прежним — они уже вызывают методы, имеющиеся в интерфейсе.
- `cmd/shorten/main.go` собирает зависимости (без изменений по сравнению с M2):
  ```go
  mem := store.NewMemoryStorage()
  s := shortener.NewShortener(mem)   // *MemoryStorage удовлетворяет Storage
  ```

### 3. Sentinel-ошибки (в `internal/store/errors.go`)
```go
package store

import "errors"

// Sentinel-ошибки хранилища.
var (
    ErrNotFound  = errors.New("link not found")
    ErrExpired   = errors.New("link expired")
    ErrDuplicate = errors.New("link code already exists")
)
```
- Использовать в `MemoryStorage`: `return nil, ErrNotFound` и т.п. (заменяют заглушки `errors.New` из M2/M4).

### 4. Кастомный тип ошибки с контекстом (в `internal/store/errors.go`)
```go
// LinkError добавляет контекст кода ссылки к обёрнутой ошибке.
type LinkError struct {
    Code string
    Err  error
}

func (e *LinkError) Error() string { return e.Code + ": " + e.Err.Error() }
func (e *LinkError) Unwrap() error { return e.Err }
```
- В `Resolve` оборачивать:
  ```go
  return nil, &LinkError{Code: code, Err: ErrNotFound}
  ```
- Это позволяет `errors.Is(err, ErrNotFound)` возвращать `true` (через `Unwrap`).

### 5. Обёртка через `%w` (в `internal/shortener/shortener.go`)
- В `Shorten` оборачивать ошибки хранилища:
  ```go
  if err := s.store.Save(link); err != nil {
      return "", fmt.Errorf("shortener: save link %q: %w", link.Code, err)
  }
  ```
- Завести sentinel-ошибку валидации в `internal/shortener` (рядом с `Validate` в `validate.go`):
  ```go
  var ErrInvalidURL = errors.New("invalid URL")
  ```
  `Validate` (из M3) возвращает `ErrInvalidURL` вместо `errors.New(...)`, чтобы её можно было проверить через `errors.Is`.

### 6. `cmd/shorten/main.go` — маппинг ошибок (в `cmd/shorten/main.go`)
- Проверять через `errors.Is` и выводить человекочитаемые сообщения согласно спеке 4.3 [00-spec.md](00-spec.md):
  ```go
  switch {
  case errors.Is(err, shortener.ErrInvalidURL):
      fmt.Fprintln(os.Stderr, "error: invalid URL")
  case errors.Is(err, store.ErrNotFound):
      fmt.Fprintln(os.Stderr, "error: not found")
  case errors.Is(err, store.ErrExpired):
      fmt.Fprintln(os.Stderr, "error: link expired")
  case errors.Is(err, store.ErrDuplicate):
      fmt.Fprintln(os.Stderr, "error: duplicate code")
  default:
      fmt.Fprintf(os.Stderr, "error: %v\n", err)
  }
  os.Exit(1)
  ```
- Все ветки возвращают ненулевой код (спека 4.2/4.3).

## Отработка тем (практика и демо)

### `any`, type assertion, type switch (в `internal/store/store.go`)
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

### Nil-интерфейс vs интерфейс с nil-значением (в `internal/store/store.go`)
- Демонстрационная функция `demoNilInterface()`:
  ```go
  var s Storage        // nil-интерфейс
  fmt.Println(s == nil) // true
  var mem *MemoryStorage = nil
  var s2 Storage = mem
  fmt.Println(s2 == nil) // false! интерфейс не nil, хотя значение nil
  ```
- Комментарий: интерфейс — это пара `(type, value)`; он равен `nil` только если обе компоненты `nil`. Вызов метода на `s2` паникует, если метод не проверяет nil-приёмник.

### Срезы и карты интерфейсов (в `internal/store/store.go`)
- Демонстрация: `[]Storage` со ссылкой на несколько реализаций (пока одна `MemoryStorage`, плюс можно добавить `NoopStorage` для теста).
- Комментарий: хранение интерфейсов в срезах/картах боксит значения — стоимость аллокации; не используйте интерфейсы без необходимости.

### Ресивер и реализация интерфейса (комментарий в `internal/store/store.go`)
- Если методы `Storage` объявлены с pointer-receiver (`func (m *MemoryStorage) Save`), то интерфейс реализует только `*MemoryStorage`, а не `MemoryStorage`.
- Демонстрация: `var s Storage = mem` (где `mem *MemoryStorage`) работает; `var s Storage = *mem` — нет.
- Комментарий: объяснить, почему `NewMemoryStorage` возвращает указатель.

### SOLID (комментарий в `internal/store/store.go`)
- **S** (SRP): `Storage` отвечает только за хранение.
- **O** (OCP): новая реализация (например, `PostgresStorage` в Главе 3) добавляется без изменения `shortener`.
- **L** (LSP): любая реализация `Storage` взаимозаменяема.
- **I** (ISP): интерфейс малый — только методы, нужные потребителю.
- **D** (DIP): `shortener` зависит от абстракции `Storage`, не от конкретики.

### `errors.As` (демонстрация в `internal/store/errors.go`)
- Показать извлечение `*LinkError`:
  ```go
  var le *store.LinkError
  if errors.As(err, &le) {
      fmt.Println("code:", le.Code)
  }
  ```

### `errors.Join` (демонстрация в `internal/store/errors.go`)
- Демонстрация агрегации ошибок: при валидации нескольких URL вернуть одну объединённую ошибку:
  ```go
  return errors.Join(err1, err2, err3)
  ```

### `defer` для ресурсов (демонстрация в `internal/store/errors.go`)
- Демонстрационную функцию `demoDeferFile()` (не основной путь `main`) открыть файл логов через `os.OpenFile` и закрыть через `defer`:
  ```go
  func demoDeferFile() error {
      f, err := os.OpenFile("shorten.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
      if err != nil { return err }
      defer f.Close() // LIFO: выполнится при выходе из demoDeferFile
      // ... запись в файл ...
      return nil
  }
  ```
- Внимание: не открывайте файл логов в основном пути `main` без явной опции — это загрязняло бы каждый запуск CLI побочным файлом. Демонстрация `defer` живёт в отдельной функции и вызывается только в smoke-тесте.
- Комментарий: `defer` выполняется по LIFO при выходе из функции; аргументы вычисляются в момент объявления `defer` (важно для `defer f.Close()`).

### `defer` в цикле и стоимость (демонстрация в `internal/store/errors.go`)
- Демонстрационная функция `demoDeferInLoop()`:
  - Показать антипаттерн: `defer` в цикле накапливает отложенные вызовы до конца функции (утечка ресурсов).
  - Правильный вариант — вынести тело цикла в отдельную функцию, чтобы `defer` срабатывал на каждой итерации.
- Комментарий: в Go 1.14+ `defer` оптимизирован для общего случая (open-coded defer), но в цикле всё равно копит вызовы.

### Модификация именованных возвращаемых значений (демонстрация в `internal/store/errors.go`)
- Демонстрация `defer` для panic-safe восстановления:
  ```go
  func safeDivide(a, b int) (result int, err error) {
      defer func() {
          if r := recover(); r != nil {
              err = fmt.Errorf("recovered: %v", r)
          }
      }()
      if b == 0 {
          panic("division by zero")
      }
      return a / b, nil
  }
  ```
- Комментарий: `recover` работает только в отложенной функции; именованные возвращаемые значения доступны для модификации в `defer`.

### `panic`/`recover` — только для нештатных ситуаций
- Комментарий в коде: `panic` не используется в штатной логике проекта; только демонстрация. Ошибки — это значения (errors as values).

## Подсказки / подводные камни
- Интерфейс с nil-значением — классический баг: проверка `if s != nil` проходит, но вызов метода паникует. Возвращайте `nil`-интерфейс, а не типизированный nil.
- Не объявляйте интерфейс «на всякий случай»; объявляйте там, где есть несколько реализаций или где нужен mock для тестов.
- Методы с value-receiver реализуют интерфейс и значением, и указателем; с pointer-receiver — только указателем.
- Боксинг интерфейсов — аллокация; в hot path избегайте `[]interface{}`.
- `any` и `interface{}` — синонимы с Go 1.18; предпочитайте `any` для читаемости.
- Малые интерфейсы (`io.Reader`, `io.Writer`) — идиома Go; не плодите «жирные» интерфейсы.
- `fmt.Errorf("...: %v", err)` не сохраняет цепочку — `errors.Is` вернёт `false`. Используйте `%w`.
- `recover` возвращает `interface{}`; приводите через type assertion или `fmt.Sprintf("%v", r)`.
- `defer` в цикле — частая утечка ресурсов (файловых дескрипторов, соединений).
- `errors.As` принимает указатель на целевую переменную: `errors.As(err, &target)`.
- Не оборачивайте sentinel-ошибку через `errors.New` — это создаёт новую ошибку без связи. Только `%w` или `&LinkError{...}`.
- `panic` в библиотечном коде — плохая практика; возвращайте ошибку. `panic` уместен для truly-неожиданных состояний (например, нарушение инварианта).

## Критерии готовности
- [ ] Интерфейс `Storage` объявлен; `MemoryStorage` реализует `Storage` (compile-time проверка `var _ Storage = (*MemoryStorage)(nil)`).
- [ ] `Shortener` зависит от `store.Storage`, не от `*MemoryStorage`; `NewShortener` принимает интерфейс.
- [ ] `main` собирает зависимости через конструкторы.
- [ ] `ErrNotFound`/`ErrExpired`/`ErrDuplicate` объявлены как sentinel; `ErrInvalidURL` объявлен в `shortener`.
- [ ] `LinkError` с `Unwrap` реализован.
- [ ] `%w` используется для обёртки; `errors.Is`/`errors.As`/`errors.Join` продемонстрированы.
- [ ] `cmd/shorten/main.go` выводит `error: invalid URL` / `error: not found` / `error: link expired` / `error: duplicate code` через `errors.Is` и возвращает ненулевой код.
- [ ] `describe` демонстрирует type switch и type assertion.
- [ ] `demoNilInterface` демонстрирует nil-интерфейс vs nil-значение.
- [ ] Комментарии по SOLID добавлены.
- [ ] `defer f.Close()` применяется в демонстрационной функции `demoDeferFile` (не в основном пути `main`).
- [ ] `demoDeferInLoop` показывает антипаттерн и исправление.
- [ ] `safeDivide` демонстрирует `panic`/`recover` с именованными возвращаемыми значениями.
- [ ] `go build ./...`, `go vet ./...`, `gofmt -l .` проходят чисто.
- [ ] Финальный smoke-сценарий (один процесс): сократить URL → получить код → разрешить код → получить исходный URL.

## Оценка времени
~14 часов (7ч интерфейсы + 7ч ошибки/defer).