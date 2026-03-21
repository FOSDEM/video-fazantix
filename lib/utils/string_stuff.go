package utils

import (
	"bufio"
	"io"
)

func NewLineScanner(r io.Reader) *bufio.Scanner {
	scanner := bufio.NewScanner(r)
	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		for i, b := range data {
			if b == '\n' || b == '\r' {
				return i + 1, data[:i], nil
			}
		}

		if atEOF && len(data) > 0 {
			return len(data), data, nil
		}

		// Request more data if empty
		return 0, nil, nil
	})
	return scanner
}
