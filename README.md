# Cargo + Go

---

## A cargo package which receives and sends arrays

### 1. Data is generated in *generator* package
### 2. Data is *batched* and sent based on timeout or batch size limit
### 3. Received data is *Inserted* into Database

---

## Project is Dockerized on the same network

---

#### Example:
``` Go
ch := make(chan []byte, 1024)

// Producer
go func() {
	for i := 0; i < 1000; i++ {
		ch <- generate()
	}
	close(ch)
}()

// Consumer(s)
for i := 0; i < 4; i++ {
	go func() {
		for obj := range ch {
			consume(obj)
		}
	}()
}
```

#### Example-2:

``` Go
objects := make(chan []byte, 1024)

// Consumer
go func() {
	for obj := range objects {
		consume(obj) // DB, file, network, etc.
	}
}()

// Producer
// --object-count = n (eg 100,000)
go func() {
	_ = generator.GenerateStream(n, objects)
	close(objects)
}()

select {}
```