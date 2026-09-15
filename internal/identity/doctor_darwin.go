package identity

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func checkExtendedPermissions(path string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	command := BackgroundCommand(ctx, "/bin/ls", "-lde", path)
	command.Env = []string{"LC_ALL=C"}
	data, err := command.Output()
	fields := strings.Fields(string(data))
	if err != nil || len(fields) == 0 {
		return fmt.Errorf("remove extended directory ACL grants before native isolation")
	}
	if strings.Contains(fields[0], "+") {
		for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n")[1:] {
			if !strings.Contains(line, " deny ") {
				return fmt.Errorf("extended ACL access grants require an operator-only configuration")
			}
		}
	}
	return nil
}
