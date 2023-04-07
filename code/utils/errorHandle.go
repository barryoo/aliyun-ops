package utils

import "fmt"

func P(msg string, err error) {
	if err != nil {
		panic(fmt.Errorf("%s: %s", msg, err))
	}
}
