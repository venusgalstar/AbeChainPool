package utils

import (
	"fmt"
	"io/ioutil"
	"runtime"
	"strconv"
	"time"
)

var panicFilename = "panic_dump"

func MyRecover() {
	if err := recover(); err != nil {
		fmt.Println(err)
		var buf [4096]byte
		n := runtime.Stack(buf[:], false)
		fmt.Printf("Stack Trace ==> %s\n", string(buf[:n]))
		fmt.Println("Recovering...")
		// Dump panic file
		_ = DumpPanicInfo(fmt.Sprintf("%v", err) + "\n" + string(buf[:]))
	}
}

func DumpPanicInfo(info string) error {
	currentTime := time.Now()
	fileSuffix := currentTime.Format("20060102150405") + "_" + strconv.FormatInt(currentTime.Unix(), 10)
	fileName := panicFilename + "_" + fileSuffix
	log.Infof("Dumping panic info to %v...", fileName)
	err := ioutil.WriteFile(fileName, []byte(info), 0666)
	if err != nil {
		log.Errorf("Unable to write panic file %v", fileName)
		return err
	}
	return nil
}
