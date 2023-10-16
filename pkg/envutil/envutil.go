// From https://github.com/reproducible-containers/repro-get/blob/v0.4.0/pkg/envutil/envutil.go

package envutil

import (
	"os"
	"strconv"
	"strings"

	"github.com/containerd/log"
)

func String(envName, defaultValue string) string {
	v, ok := os.LookupEnv(envName)
	if !ok {
		return defaultValue
	}
	return v
}

func StringSlice(envName string, defaultValue []string) []string {
	v, ok := os.LookupEnv(envName)
	if !ok {
		return defaultValue
	}
	ss := strings.Split(v, ",")
	l := len(ss)
	for i := 0; i < l; i++ {
		ss[i] = strings.TrimSpace(ss[i])
	}
	return ss
}

func Bool(envName string, defaultValue bool) bool {
	v, ok := os.LookupEnv(envName)
	if !ok {
		return defaultValue
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		log.L.WithError(err).Warnf("Failed to parse %q ($%s) as a boolean", v, envName)
		return defaultValue
	}
	return b
}
