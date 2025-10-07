package metrics

import "strings"

func FlattenName(name string) string {
	name = strings.Replace(name, " ", "_", -1)
	name = strings.Replace(name, ".", "_", -1)
	name = strings.Replace(name, "-", "_", -1)
	name = strings.Replace(name, "=", "_", -1)
	name = strings.Replace(name, "/", "_", -1)
	return name
}

func BuildFQName(names ...string) string {
	name := strings.Join(names, "_")
	return FlattenName(name)
}
