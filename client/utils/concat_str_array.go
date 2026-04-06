package utils

func ConcatStr(str *[]string) string {

	tmp := ""
	for _, s := range *str {
		tmp += s
	}
	return tmp
}
