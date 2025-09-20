package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

type Tokens struct {
	Token string `json:"token"`
}

func makeRequest(wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		time.Sleep(10 * time.Second)
		req, err := http.NewRequest("GET", "https://127.0.0.1:31411/api/auth", nil)
		if err != nil {
			fmt.Printf("%s\n", err.Error())
		}

		req.Header.Add("Authorization", `Basic `)

		client := &http.Client{Timeout: 100 * time.Second}

		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("%s\n", err.Error())
		}
		resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("Error reading response body:", err)
		}

		fmt.Printf("Body: %s\n", string(body))

		var j Tokens
		err = json.Unmarshal(body, &j)
		if err != nil {
			fmt.Println(err.Error())
		}

		fmt.Println("Token: " + string(j.Token))

		// Request #2
		req2, err := http.NewRequest("GET", "https://127.0.0.1:31411/api/auth?type=check-token", nil)
		if err != nil {
			fmt.Printf("%s\n", err.Error())
		}

		req2.Header.Add("Authorization", `Bearer `+string(j.Token))

		client2 := &http.Client{Timeout: 100 * time.Second}

		resp2, err := client2.Do(req2)
		if err != nil {
			fmt.Printf("%s\n", err.Error())
		}
		resp2.Body.Close()

		body2, err := io.ReadAll(resp2.Body)
		if err != nil {
			fmt.Println("Error reading response body:", err)
		}

		fmt.Printf("Body: %s\n", string(body2))

		var j2 Tokens
		err = json.Unmarshal(body2, &j)
		if err != nil {
			fmt.Println(err.Error())
		}
		fmt.Println("Token2: " + string(j2.Token))
	}
}

func main() {
	var wg sync.WaitGroup

	wg.Add(2)
	for i := 1; i <= 2; i++ {
		go makeRequest(&wg)
	}
	wg.Wait()
}
