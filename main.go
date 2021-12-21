package main

import (
	"fmt"
	"io/ioutil"
    "os"

	"./nbtparser"
)

func main() {
	buffer, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		fmt.Print(err)
	}

	result := nbtparser.ParseNBT(buffer, os.Args[2]=="true")
	var strBuffer string
	result.Print(&strBuffer, "")
	fmt.Print(strBuffer)
}
