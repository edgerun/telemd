package telem

import (
	"bufio"
	"os"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

// readFirstLine reads and returns the first line from the given file.
// propagates errors from os open and bufio.Scanner.
func readFirstLine(path string) (string, error) {
	file, err := os.Open(path)
	check(err)

	defer func() {
		err = file.Close()
		check(err)
	}()

	scanner := bufio.NewScanner(file)

	scanner.Scan()
	text := scanner.Text()

	return text, scanner.Err()
}
