# Задача 09. Обработка ошибок, panic и defer

## Связанные темы
- 1.10.1 Философия обработки ошибок в Go (errors as values)
- 1.10.2 Создание собственных типов ошибок (sentinel-ошибки и кастомные типы)
- 1.10.3 Оборачивание ошибок (%w) и функции errors.Is / errors.As / errors.Join
- 1.10.4 defer: семантика LIFO, момент вычисления аргументов и базовые применения
- 1.10.5 defer в цикле, модификация именованных возвращаемых значений и стоимость defer
- 1.10.6 Механика panic и recover: когда и как применять

## Цель
Заменить заглушки `errors.New` на полноценную систему ошибок проекта: sentinel-ошибки `ErrNotFound`/`ErrExpired`/`ErrDuplicate`, кастомный тип ошибки с контекстом, обёртку через `%w`, проверку через `errors.Is`/`errors.As`. Применить `defer` для закрытия ресурсов и продемонстрировать `panic`/`recover`.

## Что нужно сделать

### 1. Sentinel-ошибки (`internal/store/errors.go`)
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
- Использовать в `MemoryStorage`: `return nil, ErrNotFound` и т.п.

### 2. Кастомный тип ошибки с контекстом
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

### 3. Обёртка через `%w`
- В `Shorten` оборачивать ошибки хранилища:
  ```go
  if err := s.store.Save(link); err != nil {
      return "", fmt.Errorf("shortener: save link %q: %w", link.Code, err)
  }
  ```
- В `cmd/shorten/main.go` проверять через `errors.Is`:
  ```go
  if errors.Is(err, store.ErrNotFound) {
      fmt.Fprintln(os.Stderr, "error: not found")
      os.Exit(1)
  }
  if errors.Is(err, store.ErrExpired) {
      fmt.Fprintln(os.Stderr, "error: link expired")
      os.Exit(1)
  }
  ```

### 4. `errors.As`
- В демонстрации показать извлечение `*LinkError`:
  ```go
  var le *store.LinkError
  if errors.As(err, &le) {
      fmt.Println("code:", le.Code)
  }
  ```

### 5. `errors.Join`
- Демонстрация агрегации ошибок: при валидации нескольких URL вернуть одну объединённую ошибку:
  ```go
  return errors.Join(err1, err2, err3)
  ```

### 6. `defer` для ресурсов
- В `cmd/shorten/main.go` или демонстрационной функции открыть файл логов через `os.OpenFile` и закрыть через `defer`:
  ```go
  f, err := os.OpenFile("shorten.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
  if err != nil { ... }
  defer f.Close()
  ```
- Комментарий: `defer` выполняется по LIFO при выходе из функции; аргументы вычисляются в момент объявления `defer` (важно для `defer f.Close()`).

### 7. `defer` в цикле и стоимость (тема 1.10.5)
- Демонстрационная функция `demoDeferInLoop()`:
  - Показать антипаттерн: `defer` в цикле накапливает отложенные вызовы до конца функции (утечка ресурсов).
  - Правильный вариант — вынести тело цикла в отдельную функцию, чтобы `defer` срабатывал на каждой итерации.
- Комментарий: в Go 1.14+ `defer` оптимизирован для общего случая (open-coded defer), но в цикле всё равно копит вызовы.

### 8. Модификация именованных возвращаемых значений
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

### 9. `panic`/`recover` — только для нештатных ситуаций
- Комментарий в коде: `panic` не используется в штатной логике проекта; только демонстрация. Ошибки — это значения (errors as values).

## Требования к коду
- Все ошибки из хранилища — sentinel или кастомный тип с `Unwrap`.
- Обёртка через `%w` (не `%v`), чтобы `errors.Is`/`errors.As` работали.
- `defer` используется для закрытия ресурсов; в цикле — вынесен в функцию.
- Комментарии на русском объясняют LIFO, момент вычисления аргументов, стоимость defer, panic/recover.

## Критерии готовности
- [ ] `ErrNotFound`/`ErrExpired`/`ErrDuplicate` объявлены как sentinel.
- [ ] `LinkError` с `Unwrap` реализован.
- [ ] `%w` используется для обёртки; `errors.Is`/`errors.As`/`errors.Join` продемонстрированы.
- [ ] `defer f.Close()` применяется для файла логов.
- [ ] `demoDeferInLoop` показывает антипаттерн и исправление.
- [ ] `safeDivide` демонстрирует `panic`/`recover` с именованными возвращаемыми значениями.
- [ ] `go build ./...`, `go vet ./...`, `gofmt -l .` проходят чисто.

## Оценка времени
~7 часов.

## Подсказки / подводные камни
- `fmt.Errorf("...: %v", err)` не сохраняет цепочку — `errors.Is` вернёт `false`. Используйте `%w`.
- `recover` возвращает `interface{}`; приводите через type assertion или `fmt.Sprintf("%v", r)`.
- `defer` в цикле — частая утечка ресурсов (файловых дескрипторов, соединений).
- `errors.As` принимает указатель на целевую переменную: `errors.As(err, &target)`.
- Не оборачивайте sentinel-ошибку через `errors.New` — это создаёт новую ошибку без связи. Только `%w` или `&LinkError{...}`.
- `panic` в библиотечном коде — плохая практика; возвращайте ошибку. `panic` уместен для truly-неожиданных состояний (например, нарушение инварианта).
