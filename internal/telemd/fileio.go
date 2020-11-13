package telemd

import (
	"bufio"
	"os"
	"strconv"
)

// visitLines processes the given file line by line and applies a visitor
// function to each line. If the visitor returns false, then the method returns
// and the file is closed.
func visitLines(path string, visitor func(string) bool) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		cont := visitor(scanner.Text())
		if !cont {
			break
		}
	}

	return scanner.Err()
}

func parseInt64Array(arr []string) ([]int64, error) {
	ints := make([]int64, len(arr))
	var err error = nil

	for i := 0; i < len(arr); i++ {
		ints[i], err = strconv.ParseInt(arr[i], 10, 64)
		if err != nil {
			return ints, err
		}
	}

	return ints, err
}

// readFirstLine reads and returns the first line from the given file.
// propagates errors from os open and bufio.Scanner.
func readFirstLine(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}

	defer func() {
		_ = file.Close()
	}()

	scanner := bufio.NewScanner(file)

	scanner.Scan()
	text := scanner.Text()

	return text, scanner.Err()
}

func readLineAndParseInt(path string) (int64, error) {
	line, err := readFirstLine(path)
	if err != nil {
		return -1, err
	}
	return strconv.ParseInt(line, 10, 64)
}

func fileDirExists(filename string) bool {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return true
}

