# Задача 04. Строки: генерация кода и валидация URL

## Связанные темы
- 1.5.1 Строки, байты и руны: внутреннее представление и UTF-8
- 1.5.2 Основные операции со строками
- 1.5.3 Пакет strings: поиск, замена, разбиение
- 1.5.4 Пакет strconv: конвертация строк и чисел
- 1.5.5 Регулярные выражения: пакет regexp
- 1.5.6 Эффективная конкатенация: strings.Builder и bytes.Buffer

## Цель
Реализовать полноценную генерацию короткого кода и валидацию URL в `internal/shortener`, заменив заглушки из задачи 02. Использовать `strings.Builder`, `regexp`, `strconv`, корректно работать с UTF-8.

## Что нужно сделать

### 1. Валидация URL (`internal/shortener/validate.go`)
- Завести пакетную переменную:
  ```go
  var urlRe = regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
  ```
- Реализовать:
  ```go
  // Validate проверяет, что url является корректным HTTP(S) URL.
  func Validate(url string) error
  ```
- Возвращает ошибку при несоответствии (кастомный тип ошибки — задача 09; пока `errors.New`).
- Дополнительно через `strings`:
  - `strings.HasPrefix` для схемы,
  - `strings.Contains` для запрещённых символов,
  - `strings.ToLower` для нормализации схемы.

### 2. Генерация короткого кода (`internal/shortener/codegen.go`)
- Алфавит: `const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"`.
- Реализовать:
  ```go
  // GenerateCode возвращает случайный код длины n из алфавита alphabet.
  func GenerateCode(n int) string
  ```
- Использовать `strings.Builder` для эффективной конкатенации:
  ```go
  var b strings.Builder
  b.Grow(n)
  for i := 0; i < n; i++ {
      b.WriteByte(alphabet[randInt(len(alphabet))])
  }
  return b.String()
  ```
- Источник случайности — `math/rand` (без конкурентности — Глава 2); сид можно зафиксировать для детерминированных smoke-тестов.
- Длина по умолчанию — `defaultCodeLen = 6` (из задачи 02).

### 3. Демонстрация UTF-8 и рун
- Завести функцию `demoRunes(s string)`:
  - Показать, что `len(s)` — это байты, а не руны.
  - Итерация `for i, r := range s` даёт индекс байта и руну (`rune`).
  - Использовать `[]rune(s)` для подсчёта «символов».
- Комментарий: объяснить, что строка в Go — это неизменяемая последовательность байт, обычно UTF-8.

### 4. Использование `strconv`
- В `demoRunes` или вспомогательной функции продемонстрировать:
  - `strconv.Itoa` / `strconv.FormatInt` для чисел,
  - `strconv.Atoi` для обратного преобразования,
  - обработку ошибок `strconv.Atoi` (`_, err := strconv.Atoi(s)`).

### 5. Интеграция с `Shorten`
- В `Shorten` (задача 02) заменить заглушку генерации на `GenerateCode(defaultCodeLen)`.
- Валидация — через `Validate(longURL)`.
- Сборка итоговой короткой ссылки: `urlPrefix + code` через `strings.Builder` (не `+`).

## Требования к коду
- Для многократной конкатенации — `strings.Builder` с предзапросом `Grow`.
- `regexp.MustCompile` на уровне пакета (компилируется один раз).
- Комментарии на русском объясняют выбор `Builder` vs `+` vs `bytes.Buffer`.

## Критерии готовности
- [ ] `Validate` реализован через `regexp` и `strings`.
- [ ] `GenerateCode` использует `strings.Builder` и алфавит `[A-Za-z0-9]`.
- [ ] `demoRunes` демонстрирует разницу байт и рун.
- [ ] `strconv` использован для конвертации чисел.
- [ ] `Shorten` использует `GenerateCode` и `Validate`.
- [ ] `go build ./...`, `go vet ./...`, `gofmt -l .` проходят чисто.

## Оценка времени
~8 часов.

## Подсказки / подводные камни
- `regexp.MustCompile` паникует при ошибке паттерна — используйте только для литералов, проверенных вручную.
- `strings.Builder` не копируйте после использования — его внутренний буфер не должен разделяться.
- `len(string)` — байты, не символы; для подсчёта рун — `utf8.RuneCountInString(s)`.
- Конкатенация через `+` в цикле квадратична по аллокациям; `Builder` — линейна.
- `bytes.Buffer` уместен, когда нужно работать с байтами; для строк предпочтительнее `strings.Builder`.
- Не храните `*regexp.Regexp` в локальных переменных — компилируйте один раз на уровне пакета.
