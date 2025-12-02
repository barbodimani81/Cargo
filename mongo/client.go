package mongo

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"sync"
	"time"
)

func NewClient(uri string) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatalf("connect error: %v", err)
	}
	defer func() {
		_ = client.Disconnect(context.Background())
	}()
	return client, nil
}

func main() {
	// ----------------
	collection := client.Database("testdb").Collection("items")
	docsCh := make(chan bson.M, 100)

	start := time.Now()
	go func() {
		// 2. Use generator to produce N random items as JSON
		defer close(docsCh)
		ch := generator.Generate(*generatedCount)

		// 3. For each generated JSON payload, unmarshal and insert into Mongo
		for payload := range ch {
			var doc bson.M
			if err := json.Unmarshal(payload, &doc); err != nil {
				log.Fatalf("unmarshal error: %v", err)
			}
			docsCh <- doc
		}
	}()

	var wg sync.WaitGroup
	wg.Add(*workerCount)
	for i := 0; i < *workerCount; i++ {
		go func() {
			defer wg.Done()
			var batch []any

			for doc := range docsCh {
				batch = append(batch, doc)
				if len(batch) >= *batchLimit {
					_, err := collection.InsertMany(ctx, batch)
					if err != nil {
						log.Fatalf("insert error: %v", err)
					}
					fmt.Printf("Inserted: %d items\n", *batchLimit)
					batch = batch[:0]
				}
			}
			if len(batch) > 0 {
				if _, err := collection.InsertMany(ctx, batch); err != nil {
					log.Fatalf("final insert error: %v", err)
				}
			}
		}()
	}

	wg.Wait()
	elapsed := time.Since(start)
	fmt.Println("done in: ", elapsed)
}
