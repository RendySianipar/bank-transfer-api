package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"sync"
)

func main() {
	const numRequests = 10
	const transferAmount = 200000

	var wg sync.WaitGroup
	results := make(chan string, numRequests)

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(reqNum int) {
			defer wg.Done()

			body := []byte(fmt.Sprintf(
				`{"from_account_id":"ACC001","to_account_id":"ACC002","amount":%d}`,
				transferAmount,
			))

			req, err := http.NewRequest("POST", "http://localhost:7070/transfer", bytes.NewBuffer(body))
			if err != nil {
				results <- fmt.Sprintf("request %d: build error: %v", reqNum, err)
				return
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiMWNmMzc5NTEtOWVmYi00NDAwLTgxOTYtNGRiNTlhMTVkYjc2IiwidXNlcm5hbWUiOiJyZW5keSIsImV4cCI6MTc4OTg3MzgzNywiaWF0IjoxNzg5Nzg3NDM3fQ.MWB1tKfyAUSyzBFyEXU5JDi4bJNlP0qChrpo58IwVY0")
			req.Header.Set("Idempotency-Key", fmt.Sprintf("concurrency-test-%d", reqNum)) // UNIQUE per request - this test is about row-locking, not idempotency

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				results <- fmt.Sprintf("request %d: error: %v", reqNum, err)
				return
			}
			defer resp.Body.Close()

			results <- fmt.Sprintf("request %d: status %d", reqNum, resp.StatusCode)
		}(i)
	}

	wg.Wait()
	close(results)

	successCount := 0
	for r := range results {
		fmt.Println(r)
		// crude check - adjust based on your actual success status code
	}
	fmt.Println("done, success count:", successCount)

	if err := http.ListenAndServe(":7070", nil); err != nil {
		log.Fatal(err)
	}
}
