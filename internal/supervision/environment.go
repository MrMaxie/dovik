package supervision

import (
	"os"
	"sort"
)

func appendEnvironment(overrides map[string]string) []string {
	environment := os.Environ()
	keys := make([]string, 0, len(overrides))
	for key := range overrides {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		environment = append(environment, key+"="+overrides[key])
	}
	return environment
}
