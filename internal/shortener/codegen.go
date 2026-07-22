package shortener

import "strconv"

type CodeGenerator func(seed string) string

func newCounterGenerator(start int) CodeGenerator {
	counter := start
	return func(seed string) string {

		counter++
		return strconv.FormatInt(int64(len(seed)+counter), 36)
	}
}
