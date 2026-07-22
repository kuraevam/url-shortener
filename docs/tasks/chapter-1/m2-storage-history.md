# Веха M2. Хранилище, модель данных и история переходов

## Связанные темы
- 1.4.1 Массивы: фиксированный размер и семантика значения
- 1.4.2 Срезы (slices): создание (make, литерал, slice-выражение) и базовые операции
- 1.4.3 Тройка ptr/len/cap и связь с backing array: что копируется при передаче среза
- 1.4.4 append и переаллокация, shared backing array, утечки памяти и slices.Clip
- 1.6.1 Карты: создание, доступ, удаление элементов
- 1.6.2 Итерация по карте и порядок обхода
- 1.6.3 Внутреннее устройство map (Swiss Tables в Go 1.24+) и особенности производительности
- 1.8.1 Указатели: синтаксис и семантика
- 1.8.2 Структуры: объявление, инициализация, теги
- 1.8.3 Встраивание структур (embedding) и композиция
- 1.8.4 Методы типов: value и pointer receiver
- 1.8.5 Подводные камни при работе с указателями
- 1.8.6 Экспортируемость: публичные и приватные поля и методы

## Результат вехи
Приложение переходит с заглушки `map[string]string` из M1 на полноценное хранилище `MemoryStorage` (`map[string]*Link`): пакетные `Shorten`/`Resolve` и глобальная `map` удалены, появляются методы `*Shortener` с инъекцией хранилища. Введена финальная модель данных — `Link` (с встроенным `Audit`, `Hits`, тегом `json:"long_url"`), `Audit`, `Visit` — и методы `Link` с pointer-receiver (`IncHits`, `String`, `Touch`; `IsExpired` добавится в M4). Добавлена история переходов `[]Visit` с методами `RecordVisit`/`History`/`HistoryByCode`. CLI переписан под `NewMemoryStorage` → `NewShortener`. Smoke-цикл `shorten`/`resolve` теперь работает на `MemoryStorage`. TTL и кастомные ошибки подключаются в M4 и M5.

## Что собираем (шаг к результату)

### 1. Типы модели (в `internal/store/link.go`)
- `Audit`:
  ```go
  // Audit хранит служебные метаданные создания/изменения.
  type Audit struct {
      CreatedAt time.Time
      UpdatedAt time.Time
  }
  ```
- `Link` (финальная модель — поле `TTL` добавится в M4):
  ```go
  // Link описывает сокращённую ссылку и её метаданные.
  type Link struct {
      Code    string
      LongURL string `json:"long_url"`
      Hits    int64 // счётчик переходов
      Audit        // встраивание — поля Audit промоутятся (тема 1.8.3)
  }
  ```
- `Visit`:
  ```go
  // Visit описывает факт обращения к короткой ссылке.
  type Visit struct {
      Code      string
      VisitedAt time.Time
  }
  ```
- Инициализация `Link` в `Shorten` через литерал: `&Link{Code: code, LongURL: longURL, Audit: Audit{CreatedAt: time.Now()}}`.

### 2. `MemoryStorage` (в `internal/store/memory.go`)
```go
type MemoryStorage struct {
    links   map[string]*Link   // code → Link
    history []Visit            // из этой же вехи, п. 4
}
```
- Конструктор:
  ```go
  // NewMemoryStorage создаёт пустое хранилище в памяти.
  func NewMemoryStorage() *MemoryStorage {
      return &MemoryStorage{links: make(map[string]*Link)}
  }
  ```

### 3. Методы работы с картой (в `internal/store/memory.go`)
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
- `Resolve`: `link, ok := m.links[code]`; если `!ok` → `ErrNotFound` (кастомные ошибки — веха M5; здесь — заглушки `errors.New`).
- `Delete`: `delete(m.links, code)` — idempotent, без ошибки при отсутствии.
- `Len`: `return len(m.links)`.
- Везде, где нужно различать «ключа нет» и «нулевое значение», использовать форму `v, ok := m[k]` (comma-ok). Не использовать `m[k] == nil` (особенно важно для `*Link`).

### 4. История переходов (в `internal/store/memory.go`)
- Поле `history []Visit` уже объявлено в п. 2.
- Методы:
  ```go
  // RecordVisit добавляет факт перехода по коду в историю.
  func (m *MemoryStorage) RecordVisit(code string) error
  // History возвращает копию среза истории переходов.
  func (m *MemoryStorage) History() []Visit
  // HistoryByCode возвращает историю переходов только по заданному коду.
  func (m *MemoryStorage) HistoryByCode(code string) []Visit
  ```
- `RecordVisit` использует `append`: `m.history = append(m.history, Visit{...})`.
- `History` возвращает копию, чтобы не раскрывать внутренний срез наружу (защита от shared backing array).
- `HistoryByCode`: после фильтрации использовать `slices.Clip` для освобождения неиспользуемой ёмкости (демонстрация борьбы с утечкой памяти при долгоживущих срезах). Комментарий на русском: зачем `Clip` (сброс cap до len, чтобы GC мог освободить backing array).

### 5. Итерация по карте (в `internal/store/memory.go`)
- Реализовать метод `ForEach(fn func(code string, link *Link))`:
  ```go
  for code, link := range m.links {
      fn(code, link)
  }
  ```
- Использовать `range` с двумя переменными (`code, link`).
- Замечание: `ForEach` и итератор `All()` (веха M6) перекрывают функциональность. После M6 `ForEach` можно удалить, оставив `All()` как предпочтительный способ обхода. До M6 `ForEach` — рабочая альтернатива.

### 6. Методы `Link` с pointer-receiver (в `internal/store/link.go`)
- Все методы `Link` — pointer-receiver (согласованность):
  ```go
  // IncHits увеличивает счётчик переходов. Pointer-receiver, т.к. мутирует.
  func (l *Link) IncHits() { l.Hits++ }
  // String возвращает человекочитаемое представление. Pointer-receiver
  // выбран для согласованности: если хоть один метод Link имеет
  // pointer-receiver (IncHits), все методы типа получают
  // pointer-receiver. Это также гарантирует, что *Link реализует
  // fmt.Stringer единообразно с остальной поверхностью API.
  func (l *Link) String() string
  ```
- `String` использует `l.CreatedAt.Format(time.RFC3339)` (форматирование времени).
- Комментарий: правило — если хоть один метод требует pointer-receiver, все методы типа получают pointer-receiver для согласованности и корректной реализации интерфейсов.
- Метод `Audit` (в `internal/store/link.go`):
  ```go
  // Touch обновляет UpdatedAt. Вызывается через link.Touch() (промоутинг методов).
  func (a *Audit) Touch()
  ```
- `IsExpired` будет добавлен в M4 вместе с полем `TTL`.

### 7. Рефакторинг на методы `*Shortener` (в `internal/shortener/shortener.go`)
- Завести структуру с полем `store *store.MemoryStorage` (инъекция через конструктор):
  ```go
  // Shortener реализует сокращение и восстановление URL поверх хранилища.
  type Shortener struct {
      store *store.MemoryStorage
  }

  // NewShortener создаёт Shortener с заданным хранилищем.
  func NewShortener(s *store.MemoryStorage) *Shortener {
      return &Shortener{store: s}
  }
  ```
- **Пакетные функции `Shorten`/`Resolve` и глобальную `var store` из M1 удалить** — они заменяются методами `*Shortener`:
  ```go
  // Shorten возвращает полный короткий URL (https://s.io/<code>) для longURL.
  func (s *Shortener) Shorten(longURL string) (string, error)
  // Resolve возвращает оригинальный URL по короткому коду.
  func (s *Shortener) Resolve(code string) (string, error)
  ```
- `Shorten` вызывает `GenerateCode` (пока из M1 — заглушка, полноценная в M3), создаёт `Link` (п. 1), вызывает `s.store.Save`; возвращает `urlPrefix + code`.
- `Resolve` вызывает `s.store.Resolve` и **пробрасывает** ошибку хранилища: при `err != nil` возвращает `("", err)`; при успехе возвращает `LongURL` и вызывает `s.store.RecordVisit(code)`.
- При коллизии кода (редко) — повторная генерация (простой цикл с лимитом попыток).
- Замечание: в M5 `store *store.MemoryStorage` заменится на интерфейс `store.Storage`; сигнатуры методов `*Shortener` не изменятся.

### 8. Переписывание `cmd/shorten/main.go` под `*Shortener`
- `main` собирает зависимости и вызывает методы `*Shortener`:
  ```go
  mem := store.NewMemoryStorage()
  s := shortener.NewShortener(mem)
  // ...
  r, err := s.Shorten(url)   // или s.Resolve(code)
  ```
- Логика парсинга (`--resolve`, позиционный URL, `--help`/`-h`) и обработка ошибок переносятся из M1 без изменений — меняется только источник вызовов.
- После этой вехи приложение работает на `MemoryStorage` (а не на глобальной `map`).

## Отработка тем (практика и демо)

### Срезы: тройка ptr/len/cap (в `internal/store/memory.go`)
- Демонстрационная функция `demoSliceGrowth()` (неэкспортируемая):
  - Создать срез через `make([]int, 0, 2)`.
  - В цикле `append` 10 элементов; после каждой операции логировать `len` и `cap`.
  - Показать моменты переаллокации (рост `cap`).
- Демонстрационная функция `demoSharedBackingArray()`:
  - Создать `a := [5]int{1,2,3,4,5}` (массив — семантика значения).
  - Получить `s := a[1:4]` (slice-выражение).
  - Модифицировать `s[0]` и показать, что `a[1]` изменился (shared backing array).
  - Показать, что передача массива в функцию копирует его, а передача среза — нет.

### Карты: итерация и случайный порядок (в `internal/store/memory.go`)
- Демонстрация случайного порядка обхода в `ForEach`: добавить комментарий и smoke-проверку, что два последовательных `ForEach` могут выдавать ключи в разном порядке.

### Указатели: семантика и подводные камни (в `internal/store/link.go`)
- В `MemoryStorage` хранить `map[string]*Link` (указатели), чтобы изменения `Link` через методы были видны всем держателям ссылки.
- Демонстрационная функция `demoPointers()`:
  - Показать, что передача `Link` по значению копирует структуру (изменения не видны вызывающему).
  - Передача `*Link` — изменения видны.
  - Взятие адреса элемента map запрещено (`&m[k]`); для модификации — получить `link := m[k]`, мутировать, присвоить обратно, либо хранить `*Link`.
- Комментарий: nil-указатель на `Link` — `var l *Link; l.IncHits()` паникует; показывать безопасный вызов через `if l != nil`.

### Встраивание (embedding) — демонстрация (в `internal/store/link.go`)
- `Audit` встроен в `Link`. Демонстрация:
  - `link.CreatedAt` доступен напрямую (промоутинг полей); `link.Audit.UpdatedAt` — тоже.
  - Метод `Audit.Touch()` вызывается через `link.Touch()` (промоутинг методов).
- Комментарий: встраивание — это композиция, а не наследование; нет полиморфизма, только промоутинг полей и методов.

### Теги структур (в `internal/store/link.go`)
- Тег `json:"long_url"` добавлен к полю `LongURL` (подготовка к JSON в Главе 3).
- Комментарий: теги используются пакетами сериализации; `reflect`-чтение тега в этой вехе не требуется.

### Экспортируемость (в `internal/store/link.go`)
- Все публичные поля и методы — с заглавной буквы (`Code`, `LongURL`, `Hits`, `IncHits`, `Touch`).
- Демонстрация: пакет `internal/shortener` не может обратиться к приватному полю `internal/store`.

## Подсказки / подводные камни
- `append` может вернуть новый backing array, если `len+1 > cap`; всегда присваивайте результат.
- Срез, возвращённый без копирования, удерживает весь исходный backing array в памяти — частая причина утечек.
- Массив в Go передаётся по значению (копируется); срез — это lightweight-структура (ptr, len, cap), копируется дёшево, но указывает на тот же массив.
- `slices.Clip` доступен с Go 1.20+; для обнуления ёмкости используйте `slices.Clip(s)`.
- Не используйте `append` к срезу, полученному как аргумент функции, без возврата — вызывающий код не увидит новую ёмкость.
- `nil`-карта доступна на чтение (возвращает нулевое значение), но `nil`-карта при записи паникует — всегда `make`.
- Порядок обхода `map` не детерминирован; не полагайтесь на него в логике.
- В Go 1.24+ внутреннее устройство map заменено на Swiss Tables — производительность улучшилась, но семантика API не изменилась.
- Не берите адрес элемента map (`&m[k]`) — он может стать невалидным после переаллокации.
- `delete` на отсутствующем ключе — no-op, не паникует.
- Итерация с модификацией карты во время `range` — поведение не определено; избегайте.
- Нельзя взять адрес элемента map: `&m[k]` — ошибка компиляции.
- Смешивание value- и pointer-receiver для одного типа — плохая практика (мешает реализации интерфейсов, веха M5).
- `nil`-указатель можно вызвать для метода с pointer-receiver; метод обязан сам проверить `l == nil`, если это возможно.
- Копирование структуры с мьютексом через value-receiver — ошибка (`go vet` поймает `copylocks`); мьютексы появятся в Главе 2.
- Теги — строки, не проверяются компилятором; опечатки обнаруживаются только потребителем (например, `encoding/json`).

## Критерии готовности
- [ ] `Link`, `Audit`, `Visit` объявлены; `Audit` встроен в `Link`; тег `json:"long_url"` добавлен.
- [ ] `MemoryStorage` использует `map[string]*Link`; конструктор `NewMemoryStorage`.
- [ ] `Save`/`Resolve`/`Delete`/`Len`/`ForEach` реализованы; проверка существования — через comma-ok.
- [ ] `MemoryStorage` хранит `history []Visit`; методы `RecordVisit`/`History`/`HistoryByCode` реализованы; `slices.Clip` применён в `HistoryByCode`.
- [ ] Все методы `Link` используют pointer-receiver с обоснованием выбора; `IncHits`/`String`/`Touch` реализованы.
- [ ] Пакетные `Shorten`/`Resolve` и глобальная `map` удалены; реализованы методы `*Shortener` с `MemoryStorage`; `Resolve` вызывает `RecordVisit`.
- [ ] `cmd/shorten/main.go` переписан под `*Shortener` (`NewMemoryStorage` → `NewShortener`).
- [ ] `demoSliceGrowth`/`demoSharedBackingArray`/`demoPointers` написаны и вызываются.
- [ ] `ForEach` демонстрирует итерацию и случайный порядок.
- [ ] Правила экспортируемости соблюдены.
- [ ] `go build ./...`, `go vet ./...`, `gofmt -l .` проходят чисто.
- [ ] Smoke-проверка: `go run ./cmd/shorten https://example.com` печатает `https://s.io/<code>`; `--resolve <code>` восстанавливает URL (данные хранятся в `MemoryStorage`).

## Оценка времени
~19 часов (5ч карты + 6ч срезы + 8ч структуры/методы).