# Веха M6. Дженерики и итераторы (расширение хранилища)

## Связанные темы
- 1.12.1 Введение в дженерики: параметры типа и ограничения (constraints)
- 1.12.2 Пакет constraints и создание собственных ограничений (union, ~T)
- 1.12.3 Рекурсивные ограничения типов
- 1.12.4 Практические паттерны применения дженериков (и когда они избыточны)
- 1.13.1 Range-over-func итератор: концепция, Seq и Seq2, использование
- 1.13.2 Пакеты iter, slices, maps для работы с последовательностями

## Результат вехи
Хранилище расширено обобщёнными утилитами и итераторами: в нейтральном пакете `internal/xslices` появляются `Filter`/`Map`/`Reduce`/`Sum` (с ограничением `Number`); `HistoryByCode` переписан через `xslices.Filter`; на `MemoryStorage` реализован ленивый итератор `All() iter.Seq2[*Link, error]` (range-over-func) и композиция `Active() iter.Seq[*Link]` (только неистёкшие ссылки). Эти возможности не обязательны для базового цикла `shorten`/`resolve` (приложение работало с M5), но должны компилироваться и проходить `go vet`.

## Что собираем (шаг к результату)

### 1. Обобщённые утилиты (в `internal/xslices/xslices.go`)
```go
// Filter возвращает новый срез, содержащий только те элементы items,
// для которых pred(item) == true. Исходный срез не модифицируется.
func Filter[T any](items []T, pred func(T) bool) []T {
    out := make([]T, 0, len(items))
    for _, v := range items {
        if pred(v) {
            out = append(out, v)
        }
    }
    return out
}
```
- `Map[T, U any](items []T, fn func(T) U) []U` — преобразование каждого элемента.
- `Reduce[T, U any](items []T, init U, fn func(U, T) U) U` — свёртка.
- Пакет `internal/xslices` нейтрален: на него ссылаются и `internal/shortener`, и `internal/store`, не создавая цикла импортов (см. раздел 3.2 [00-spec.md](00-spec.md)).

### 2. Собственное ограничение с `~T` (в `internal/xslices/xslices.go`)
```go
type Number interface {
    ~int | ~int64 | ~float64
}
// Sum возвращает сумму числовых элементов.
func Sum[T Number](items []T) T
```

### 3. Применение в проекте (в `internal/store/memory.go`)
- `HistoryByCode(code string) []Visit` (из M2) переписать через `xslices.Filter`:
  ```go
  func (m *MemoryStorage) HistoryByCode(code string) []Visit {
      return xslices.Filter(m.history, func(v Visit) bool { return v.Code == code })
  }
  ```
- Фильтрация активных (не истёкших) ссылок через `xslices.Filter[*Link]` + `IsExpired` (демонстрация):
  ```go
  active := xslices.Filter(allLinks, func(l *Link) bool { return !l.IsExpired(time.Now()) })
  ```

### 4. Метод `All()` (в `internal/store/iter.go`)
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
- Метод `All()` уже в интерфейсе `Storage` (веха M5); здесь — конкретная реализация (заменяет заглушку из M5).

### 5. Итератор-фильтр (композиция, в `internal/store/iter.go`)
```go
// Active возвращает итератор по неистёкшим ссылкам.
func (m *MemoryStorage) Active() iter.Seq[*Link] {
    return func(yield func(*Link) bool) {
        for link, err := range m.All() {
            if err != nil {
                // В MemoryStorage err всегда nil; зарезервировано для будущих хранилищ.
                return
            }
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

## Отработка тем (практика и демо)

### Рекурсивные ограничения (в `internal/xslices/xslices.go`)
- Демонстрация ограничения, ссылающегося на себя:
  ```go
  type Stringer interface {
      String() string
  }
  // JoinStrings объединяет элементы, реализующих Stringer.
  func JoinStrings[T Stringer](items []T) string
  ```
- Комментарий: рекурсивные ограничения (через `~` и интерфейсы, ссылающиеся на себя) — продвинутая тема; в проекте достаточно базовых.

### Демонстрация `Number` с `~T` (в `internal/xslices/xslices.go`)
- Тип `type Counter int64`, удовлетворяющий `~int64`, передаётся в `Sum` (без `~` это не сработало бы).

### Демонстрация `Map`/`Reduce` (в `internal/store/memory.go`)
- Использовать в демонстрации: например, `Map` истории переходов в срез кодов.

### Когда дженерики избыточны (комментарий в `internal/xslices/xslices.go`)
- Комментарий: дженерики уместны, когда алгоритм не зависит от типа (`Filter`, `Map`, `Reduce`, `Sum`). Если есть одна конкретная операция над одним типом — обычная функция проще и читаемее. Не вводите дженерики «для красоты».

### Использование `range-over-func` (в `internal/store/iter.go`)
- Демонстрационная функция `printAll(s *MemoryStorage)`:
  ```go
  for link, err := range s.All() {
      if err != nil { ... }
      fmt.Println(link.Code, link.LongURL)
  }
  ```
- Демонстрация прерывания: `break` после 3-й ссылки — `yield` получает `false`, итератор корректно останавливается.

### Пакет `slices` (в `internal/store/iter.go`)
- `slices.Collect` принимает `iter.Seq[T]`, а `All()` возвращает `iter.Seq2[*Link, error]` — напрямую передать нельзя. Адаптер `Seq2`→`Seq` (отбрасывая ошибку, т.к. в `MemoryStorage` она всегда `nil`):
  ```go
  linksSeq := func(yield func(*Link) bool) {
      for link, err := range s.All() {
          if err != nil { return }
          if !yield(link) { return }
      }
  }
  all := slices.Collect(linksSeq) // []*Link
  ```
- `slices.SortedFunc` для сортировки ссылок по `Code`:
  ```go
  sorted := slices.SortedFunc(slices.Values(all), func(a, b *Link) int {
      return strings.Compare(a.Code, b.Code)
  })
  ```
- Комментарий: `slices.Collect` собирает `iter.Seq[T]` в `[]T`; для `Seq2` нужен адаптер.

### Пакет `maps` (в `internal/store/iter.go`)
- Демонстрация: получить ключи карты через `maps.Keys(m.links)` (возвращает `iter.Seq[string]` в Go 1.23+):
  ```go
  for code := range maps.Keys(m.links) {
      fmt.Println(code)
  }
  ```
- `maps.Values` — итератор по значениям.

## Подсказки / подводные камни
- `any` — алиас `interface{}`; используйте как ограничение по умолчанию.
- `~T` разрешает типы с тем же underlying-типом (например, `type Counter int64`); без `~` — только сам `int64`.
- Не плодите дженерики там, где достаточно `[]*Link` и обычной функции.
- Инстанциация `Filter[Visit](...)` явная, но часто выводится автоматически — `Filter(items, pred)`.
- `comparable` — встроенное ограничение для типов, поддерживающих `==`/`!=` (нужно для ключей map).
- Цикл импортов: `Filter` размещён в `internal/xslices` именно чтобы `internal/store` мог его использовать без зависимости от `internal/shortener` (см. раздел 3.2 [00-spec.md](00-spec.md)). Не помещайте обобщённые утилиты в пакет-потребитель, если их импортирует другой пакет с другим направлением зависимости.
- `iter.Seq2[V, error]` — идиома для итераторов с возможной ошибкой; ошибка отдаётся через второе значение `yield`.
- `yield` возвращает `bool`: `false` означает, что потребитель вызвал `break` — нужно немедленно вернуть управление.
- Итераторы ленивы: тело не выполняется до начала `range`; каждый шаг — это вызов `yield` потребителем.
- `slices.Collect` принимает `iter.Seq[T]`, не `Seq2`; для `Seq2` используйте адаптер или ручной цикл.
- `maps.Keys`/`maps.Values` в Go 1.23+ возвращают `iter.Seq`, а не срез (старые версии возвращали срез).
- Не храните итератор и не переиспользуйте после исчерпания — он однопроходный (если не реализован как re-iterable).
- Требуется `go 1.23+` для `range-over-func`; в `go.mod` указано `go 1.24` (из M1).

## Критерии готовности
- [ ] `Filter[T]` реализован и используется в `HistoryByCode` и фильтрации ссылок.
- [ ] `Map` и `Reduce` реализованы.
- [ ] Ограничение `Number` с `~int | ~int64 | ~float64` и `Sum[T Number]` продемонстрированы.
- [ ] `JoinStrings[T Stringer]` демонстрирует интерфейсное ограничение.
- [ ] Комментарий об избыточности дженериков добавлен.
- [ ] `All() iter.Seq2[*Link, error]` реализован на `MemoryStorage` (заменяет заглушку из M5).
- [ ] `range-over-func` используется для обхода `All()`; `break` корректно прерывает итератор.
- [ ] `slices.Collect`/`slices.SortedFunc` применены.
- [ ] `maps.Keys`/`maps.Values` продемонстрированы.
- [ ] `Active() iter.Seq[*Link]` реализован как композиция.
- [ ] Интерфейс `Storage` реализуется `MemoryStorage` (compile-time проверка из M5 проходит).
- [ ] `go build ./...`, `go vet ./...`, `gofmt -l .` проходят чисто.

## Оценка времени
~10 часов (5ч дженерики + 5ч итераторы).