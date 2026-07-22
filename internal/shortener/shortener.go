// Пакет shortener реализует логику сокращения и восстановления URL.
package shortener

import (
	"errors"
	"strings"
)

const (
	defaultCodeLen = 6
	urlPrefix      = "https://s.io/"
)

var (
	counterGenerator CodeGenerator
)

func init() {
	counterGenerator = newCounterGenerator(0)
}

var store = map[string]string{}

var ErrInvalidURL = errors.New("invalid URL")
var ErrNotFound = errors.New("not found")

type schemeKind int

const (
	schemeHTTP schemeKind = iota
	schemeHTTPS
	schemeOther
)

// Shorten возвращает полный короткий URL (https://s.io/<code>) для longURL.
func Shorten(longURL string) (string, error) {

	if err := Validate(longURL); err != nil {
		return "", err
	}

	code := counterGenerator(longURL)

	shortURL := urlPrefix + code

	store[code] = longURL

	return shortURL, nil
}

// Resolve возвращает оригинальный URL по короткому коду.
func Resolve(code string) (string, error) {

	longUrl, ok := store[code]

	if !ok {
		return "", ErrNotFound
	}

	return longUrl, nil
}

func Validate(url string) error {
	hasValid := strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")

	if !hasValid {
		return ErrInvalidURL
	}

	return nil
}

func joinURLs(parts ...string) string {
	return strings.Join(parts, "/")
}
