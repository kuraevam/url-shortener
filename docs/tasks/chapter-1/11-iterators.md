# Задача 11. Итераторы: `range-over-func` и `All()`

## Связанные темы
- 1.13.1 Range-over-func итераторы: концепция, Seq и Seq2, использование
- 1.13.2 Пакеты iter, slices, maps для работы с последовательностями

## Цель
Реализовать метод `All()` на `MemoryStorage`, возвращающий итератор `iter.Seq2[*Link, error]`, и использовать его в `range-over-func`. Применить пакеты `iter`, `slices`, `maps` для работы с последовательностями.

## Что нужно сделать

### 1. Метод `All()` (`internal/store/iter.go`)
```go
import "iter"

// All возвращает итератор по всем хранимым ссылкам. Ошибка (второе
// значение Seq2) зарезервирована для будущих хранилищ (например, БД),
// где чтение может завершиться ошибкой; в MemoryStorage всегда nil.
func (m *MemoryStorage) All() iter.Seq2[*Link, error] {
    return func(yield func(*Link, error) bool) {
        for _, link := range m.links {
            if !yield(link, nil) {
                return
            }
        }
    }
}
```
- `iter.Seq2[*Link, error]` — функция, принимающая `yield`; если `yield` возвращает `false`, итерация прерывается (вызов `break` в `range`).

### 2. Использование `range-over-func`
- В демонстрационной функции `printAll(s *MemoryStorage)`:
  ```go
  for link, err := range s.All() {
      if err != nil { ... }
      fmt.Println(link.Code, link.LongURL)
  }
  ```
- Демонстрация прерывания: `break` после 3-й ссылки — `yield` получает `false`, итератор корректно останавливается.

### 3. Пакет `slices` (тема 1.13.2)
- Собрать все ссылки в срез через итератор:
  ```go
  var all []*Link
  for link := range slices.Collect(s.All()) {
      _ = link
  }
  // или напрямую:
  links := slices.Collect(func(yield func(*Link) bool) {
      for link, _ := range s.All() {
          if !yield(link) { return }
      }
  })
  ```
  (Упростить до `slices.AppendSeq`/`slices.Collect` где возможно.)
- Использовать `slices.SortedFunc` для сортировки ссылок по `Code`.
- Комментарий: `slices.Collect` собирает `iter.Seq[T]` в `[]T`; для `Seq2` нужна адаптация.

### 4. Пакет `maps` (тема 1.13.2)
- Демонстрация: получить ключи карты через `maps.Keys(m.links)` (возвращает итератор `iter.Seq[string]` в Go 1.23+):
  ```go
  for code := range maps.Keys(m.links) {
      fmt.Println(code)
  }
  ```
- `maps.Values` — итератор по значениям.

### 5. Итератор-фильтр (композиция)
- Реализовать вспомогательный итератор, возвращающий только активные (не истёкшие) ссылки:
  ```go
  // Active возвращает итератор по неистёкшим ссылкам.
  func (m *MemoryStorage) Active() iter.Seq[*Link] {
      return func(yield func(*Link) bool) {
          for link, _ := range m.All() {
              if link.IsExpired(time.Now()) {
                  continue
              }
              if !yield(link) {
                  return
              }
          }
      }
  }
  ```
- Использовать в `range`:
  ```go
  for link := range s.Active() {
      fmt.Println(link.Code)
  }
  ```

### 6. Интеграция с интерфейсом `Storage`
- Метод `All() iter.Seq2[*Link, error]` уже в интерфейсе `Storage` (задача 08). Убедиться, что `MemoryStorage` реализует его корректно (compile-time проверка `var _ Storage = (*MemoryStorage)(nil)`).

## Требования к коду
- Используется `iter.Seq`/`iter.Seq2` и `range-over-func` (Go 1.23+; в `go.mod` указать `go 1.24`).
- Итераторы ленивые: элементы вычисляются по требованию `yield`.
- `slices.Collect`/`slices.SortedFunc`/`maps.Keys`/`maps.Values` применены в демонстрации.
- Комментарии на русском объясняют концепцию `yield`, прерывание, ленивость.

## Критерии готовности
- [ ] `All() iter.Seq2[*Link, error]` реализован на `MemoryStorage`.
- [ ] `range-over-func` используется для обхода `All()`.
- [ ] `break` корректно прерывает итератор.
- [ ] `slices.Collect`/`slices.SortedFunc` применены.
- [ ] `maps.Keys`/`maps.Values` продемонстрированы.
- [ ] `Active() iter.Seq[*Link]` реализован как композиция.
- [ ] Интерфейс `Storage` включает `All()` и реализуется `MemoryStorage`.
- [ ] `go build ./...`, `go vet ./...`, `gofmt -l .` проходят чисто.

## Оценка времени
~5 часов.

## Подсказки / подводные камни
- `iter.Seq2[V, error]` — идиома для итераторов с возможной ошибкой; ошибка отдаётся через второе значение `yield`.
- `yield` возвращает `bool`: `false` означает, что потребитель вызвал `break` — нужно немедленно вернуть управление.
- Итераторы ленивы: тело не выполняется до начала `range`; каждый шаг — это вызов `yield` потребителем.
- `slices.Collect` принимает `iter.Seq[T]`, не `Seq2`; для `Seq2` используйте адаптер или ручной цикл.
- `maps.Keys`/`maps.Values` в Go 1.23+ возвращают `iter.Seq`, а не срез (старые версии возвращали срез).
- Не храните итератор и не переиспользуйте после исчерпания — он однопроходный (если не реализован как re-iterable).
- Требуется `go 1.23+` для `range-over-func`; в `go.mod` указано `go 1.24` (из задачи 01).
