package telem

import (
	"bufio"
	"io/ioutil"
	"os"
	"strconv"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func ListFilterDir(dirname string, predicate func(info os.FileInfo) bool) []string {
	dir, err := ioutil.ReadDir(dirname)
	check(err)

	files := make([]string, 0)

	for _, f := range dir {
		if predicate(f) {
			files = append(files, f.Name())
		}
	}

	return files
}

func ParseInt64Array(arr []string) ([]int64, error) {
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

// ReadFirstLine reads and returns the first line from the given file.
// propagates errors from os open and bufio.Scanner.
func ReadFirstLine(path string) (string, error) {
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

func ReadLineAndParseInt(path string) (int64, error) {
	line, err := ReadFirstLine(path)
	if err != nil {
		return -1, err
	}
	return strconv.ParseInt(line, 10, 64)
}
