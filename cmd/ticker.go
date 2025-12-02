package main

import (
	"fmt"
	"time"
)

func main() {
	t := time.NewTicker(time.Second)

	for x := range t.C {

		fmt.Println("called")
		fmt.Println(x)
		t.Stop()
	}
}
