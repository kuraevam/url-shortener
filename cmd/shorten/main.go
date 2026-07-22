package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/kuraevam/url-shortener/internal/shortener"
)

var (
	code string
	url  string
	help bool
)

func init() {
	flag.Usage = func() {
		fmt.Println(`Usage:
	shorten <longURL>           Создать короткий код для longURL, напечатать https://s.io/<code>
	shorten --resolve <code>    Напечатать оригинальный URL по короткому коду
	shorten --help | -h         Показать эту справку`)
	}

	flag.StringVar(&code, "resolve", "", "")
	flag.BoolVar(&help, "help", false, "")
}

func main() {

	flag.Parse()

	if help {
		flag.Usage()
		os.Exit(0)
	}

	url = flag.Arg(0)
	if url != "" {
		r, err := shortener.Shorten(url)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(r)
		os.Exit(0)
	}

	if code != "" {
		r, err := shortener.Resolve(code)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(r)
		os.Exit(0)
	}

	flag.Usage()
	os.Exit(1)

}
