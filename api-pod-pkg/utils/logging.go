package utils

import (
	"bufio"
	"io"
	"log"
)

func StartLogging(reader io.ReadCloser, isError bool) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		m := scanner.Text()
		if isError {
			log.Println("[ERROR]", m)
		} else {
			log.Println(m)
		}
	}
}
