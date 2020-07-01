package telemd

import (
	"os/exec"
	"strings"
)

// executes the given command and returns the output splitted by \n
// removes the last line if it is empty
func execute(program string) ([]string, error) {
	cmd := exec.Command(program)

	if output, err := cmd.Output(); err != nil {
		return []string{}, err
	} else {
		outputs := string(output)
		split := strings.Split(outputs, "\n")

		if len(split[len(split)-1]) == 0 {
			return split[:len(split)-1], nil
		}

		return split, nil
	}
}
