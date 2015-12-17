package uuid

import (
	"fmt"
	"os/exec"
	"strings"
)

// New shells out to `uuidgen` to get a new UUID
func New(prefix string) (string, error) {
	cmd := exec.Command("uuidgen")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("Error generating uuid: %s", err)
	}

	uuid := string(out)
	uuid = strings.Trim(uuid, " \n")

	if prefix == "" {
		return uuid, nil
	}

	return fmt.Sprintf("%s-%s", prefix, uuid), nil
}
