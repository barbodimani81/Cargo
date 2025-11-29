package generator

import (
	"encoding/json"
	"log"
	"math/rand"
)

type Item struct {
	ID     int  `json:"id"`
	Age    int  `json:"age"`
	Status bool `json:"status"`
}

func ItemGenerator(n int) (<-chan []byte, error) {
	ch := make(chan []byte)

	go func() {
		defer close(ch)
		for i := 0; i < n; i++ {
			item := Item{
				ID:     rand.Intn(1000000),
				Age:    rand.Intn(80),
				Status: rand.Intn(2) == 1,
			}
			obj, err := json.Marshal(item)
			if err != nil {
				log.Fatalf("cannot marshal: %v", err)
			}
			ch <- obj
		}
	}()
	return ch, nil
}
