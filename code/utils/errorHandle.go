package utils

import "fmt"

func E(msg string, err error) ([]byte, error) {
	if err != nil {
		panic(fmt.Errorf("%s: %s", msg, err))
	}
	return nil, nil
}
