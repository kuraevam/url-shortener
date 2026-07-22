# Веха M3. Генерация кода и валидация URL

## Связанные темы
- 1.5.1 Строки, байты и руны: внутреннее представление и UTF-8
- 1.5.2 Основные операции со строками
- 1.5.3 Пакет strings: поиск, замена, разбиение
- 1.5.4 Пакет strconv: конвертация строк и чисел
- 1.5.5 Регулярные выражения: пакет regexp
- 1.5.6 Эффективная конкатенация: strings.Builder и bytes.Buffer

## Результат вехи
Заглушки из M1/M2 заменены на полноценную реализацию: `GenerateCode` генерирует случайный код длины 6 из алфавита `[A-Za-z0-9]` через `strings.Builder`; `Validate` проверяет URL через `regexp` + `strings`. `Shorten` использует `GenerateCode` и `Validate`, собирая итоговую короткую ссылку через `strings.Builder` (не `+`). Приложение теперь выдаёт настоящие случайные короткие коды и корректно отвергает невалидные URL.

## Что собираем (шаг к результату)

### 1. Валидация URL (в `internal/shortener/validate.go`)
- Перенести `Validate` из `shortener.go` (M1) в `validate.go`.
- Завести пакетную переменную:
  ```go
  var urlRe = regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
  ```
- Реализовать:
  ```go
  // Validate проверяет, что url является корректным HTTP(S) URL.
  func Validate(url string) error
  ```
- Возвращает ошибку при несоответствии (кастомный тип ошибки — веха M5; пока `errors.New`).
- Дополнительно через `strings`:
  - `strings.HasPrefix` для схемы,
  - `strings.Contains` для запрещённых символов,
  - `strings.ToLower` для нормализации схемы.

### 2. Генерация короткого кода (в `internal/shortener/codegen.go`)
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
      b.WriteByte(alphabet[rand.Intn(len(alphabet))])
  }
  return b.String()
  ```
- Источник случайности — `math/rand` (без конкурентности — Глава 2). Начиная с Go 1.20 глобальный генератор сидируется автоматически; для детерминированных smoke-тестов можно явно вызвать `rand.Seed(...)` (или использовать `math/rand/v2` с `rand.N[int](...)` в Go 1.22+).
- Длина по умолчанию — `defaultCodeLen = 6` (из M1).

### 3. Интеграция с `Shorten`
- В `Shorten` (метод `*Shortener` из M2) заменить заглушку генерации на `GenerateCode(defaultCodeLen)`.
- Валидация — через `Validate(longURL)`. На данном этапе `Validate` возвращает `errors.New("invalid URL")`; в вехе M5 эта заглушка заменяется на sentinel `ErrInvalidURL`, проверяемый через `errors.Is`.
- Сборка итоговой короткой ссылки: `urlPrefix + code` через `strings.Builder` (не `+`). `Shorten` возвращает **полный короткий URL** `https://s.io/<code>` (согласовано со спекой 4.1).

## Отработка тем (практика и демо)

### Демонстрация UTF-8 и рун (в `internal/shortener/validate.go`)
- Завести функцию `demoRunes(s string)`:
  - Показать, что `len(s)` — это байты, а не руны.
  - Итерация `for i, r := range s` даёт индекс байта и руну (`rune`).
  - Использовать `[]rune(s)` для подсчёта «символов».
- Комментарий: объяснить, что строка в Go — это неизменяемая последовательность байт, обычно UTF-8.

### Использование `strconv` (в `internal/shortener/validate.go`)
- В `demoRunes` или вспомогательной функции продемонстрировать:
  - `strconv.Itoa` / `strconv.FormatInt` для чисел,
  - `strconv.Atoi` для обратного преобразования,
  - обработку ошибок `strconv.Atoi` (`_, err := strconv.Atoi(s)`).

## Подсказки / подводные камни
- `regexp.MustCompile` паникует при ошибке паттерна — используйте только для литералов, проверенных вручную.
- `strings.Builder` не копируйте после использования — его внутренний буфер не должен разделяться.
- `len(string)` — байты, не символы; для подсчёта рун — `utf8.RuneCountInString(s)`.
- Конкатенация через `+` в цикле квадратична по аллокациям; `Builder` — линейна.
- `bytes.Buffer` уместен, когда нужно работать с байтами; для строк предпочтительнее `strings.Builder`.
- Не храните `*regexp.Regexp` в локальных переменных — компилируйте один раз на уровне пакета.

## Критерии готовности
- [ ] `Validate` реализован через `regexp` и `strings` (в `validate.go`).
- [ ] `GenerateCode` использует `strings.Builder` и алфавит `[A-Za-z0-9]` (в `codegen.go`).
- [ ] `Shorten` использует `GenerateCode` и `Validate`; сборка URL через `strings.Builder`.
- [ ] `demoRunes` демонстрирует разницу байт и рун.
- [ ] `strconv` использован для конвертации чисел.
- [ ] `go build ./...`, `go vet ./...`, `gofmt -l .` проходят чисто.

## Оценка времени
~8 часов.