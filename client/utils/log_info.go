package utils

import "fmt"

func LogPrefix(prefix string, message string) {
	fmt.Printf("[ %s ]: %s", prefix, message)

}
