package logs

import (
	"fmt"
	"log"
	"runtime"
	"strings"
)

var Mode bool = false

func init() {
	log.SetPrefix("Agent ")
}

func Info(v ...interface{}) {
	f := getShortFile()
	v = append([]interface{}{f}, v...)
	log.Println(v...)
}

func Debug(v ...interface{}) {
	if Mode {
		f := getShortFile()
		v = append([]interface{}{f}, v...)
		log.Println(v...)
	}
}

func Assert(err error, context string) {
	if err != nil {
		f := getShortFile()
		log.Fatal(f, context+": ", err)
	}
}

func getShortFile() string {
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		file = "/???/???"
		line = 0
	}
	s := strings.Split(file, "/")
	l := len(s)
	f := fmt.Sprintf("%s.%s:%d", s[l-2], s[l-1], line)
	return f
}
