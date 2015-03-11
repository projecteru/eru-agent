package logs

import (
	"log"
)

var Mode bool = false

func init() {
	log.SetPrefix("Agent ")
}

func Info(v ...interface{}) {
	log.Println(v...)
}

func Debug(v ...interface{}) {
	if Mode {
		log.Println(v...)
	}
}

func Assert(err error, context string) {
	if err != nil {
		log.Fatal(context+": ", err)
	}
}
