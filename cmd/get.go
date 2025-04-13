package cmd

import (
	"encoding/json"
	"fmt"
	"quaycli/internal/utils"
	"sync"
)

type Get struct {
	*utils.Config //get all multipule values
	Repos         string
}

// finished !!
func (g Get) Organization() {
	fmt.Println("Getting organization data for you ❤")

	var msg map[string]any

	type localStruct struct {
		Storage_precent float64
		Repos           []string
	}

	var org localStruct

	if g.Region == "" {
		g.Region = "metzuda"
	}

	url := utils.GenUrl(fmt.Sprintf("organization/%s", g.Organizations))

	//do request
	body, err, status := utils.Req(url, "GET", g.Token, nil)
	if err != nil && status > 290 {
		fmt.Println("status:", status)
		fmt.Println(err)
		return
	}

	_ = json.Unmarshal(body, &msg)

	if usage, ok := msg["quota_report"].(map[string]any); ok {
		storage := msg["quota_report"].(map[string]any)["configured_quota"].(float64)
		org.Storage_precent = (usage["quota_bytes"].(float64) / storage) * 100
		fmt.Println("storage usage precent: ", org.Storage_precent)
	} else {
		fmt.Println("You dont have permission to see storage")
	}

	url = utils.GenUrl(fmt.Sprintf("repository?public=true&namespace=%s", g.Organizations))

	body, err, status = utils.Req(url, "GET", g.Token, nil)
	if err != nil && status > 290 {
		fmt.Println("status:", status)
		fmt.Println(err)
		return
	}

	_ = json.Unmarshal(body, &msg)

	if _, ok := msg["repositories"].([]any); !ok {
		fmt.Println("You dont have permission to see repos")
		return
	}

	for _, repo := range msg["repositories"].([]any) {
		org.Repos = append(org.Repos, repo.(map[string]any)["name"].(string))
	}

	if _, ok := msg["next_page"].(string); ok {
		for msg["next_page"].(string) != "" {

			url = utils.GenUrl(fmt.Sprintf("repository?public=true&namespace=%s&next_page=%s", g.Organizations, msg["next_page"].(string)))

			msg["next_page"] = ""

			body, err, status = utils.Req(url, "GET", g.Token, nil)
			if err != nil && status > 290 {
				fmt.Println("status:", status)
				fmt.Println(err)
				return
			}

			_ = json.Unmarshal(body, &msg)

			for _, repo := range msg["repositories"].([]any) {
				org.Repos = append(org.Repos, repo.(map[string]any)["name"].(string))
			}
		}
	}

	fmt.Println("\nREPOS:")
	for _, repo := range org.Repos {
		fmt.Println(repo)
	}

}

// finished !!
func (g Get) Repo() {
	fmt.Println("Getting repo data for you ❤")

	// Buffered channels with even larger capacity
	resultChan := make(chan []string, 100)
	errorChan := make(chan error, 100)

	// Increase max concurrent workers further
	maxWorkers := 50
	sem := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup

	// Get first page to check has_additional
	url := utils.GenUrl(fmt.Sprintf("repository/%s/%s/tag?includeTags=true&onlyActiveTags=true&page=1&limit=100",
		g.Organizations, g.Repos))

	body, err, status := utils.Req(url, "GET", g.Token, nil)
	if err != nil || status > 290 {
		fmt.Println("Error:", err)
		return
	}

	var firstPageMsg map[string]any
	if err := json.Unmarshal(body, &firstPageMsg); err != nil {
		fmt.Println("JSON error:", err)
		return
	}

	// Process first page
	if tags, ok := firstPageMsg["tags"].([]any); ok && len(tags) > 0 {
		pageTags := make([]string, 0, len(tags))
		for _, tag := range tags {
			if name, ok := tag.(map[string]any)["name"].(string); ok {
				pageTags = append(pageTags, name)
			}
		}
		resultChan <- pageTags
	}

	// Check if we need to fetch more pages
	hasAdditional, ok := firstPageMsg["has_additional"].(bool)
	if !ok || !hasAdditional {
		close(resultChan)
		close(errorChan)
		return
	}

	// Start workers for remaining pages concurrently
	page := 2
	for hasAdditional {
		for i := 0; i < maxWorkers && hasAdditional; i++ {
			wg.Add(1)
			sem <- struct{}{} // Acquire semaphore

			go func(pageNum int) {
				defer wg.Done()
				defer func() { <-sem }() // Release semaphore

				url := utils.GenUrl(fmt.Sprintf("repository/%s/%s/tag?includeTags=true&onlyActiveTags=true&page=%d&limit=100",
					g.Organizations, g.Repos, pageNum))

				body, err, status := utils.Req(url, "GET", g.Token, nil)
				if err != nil || status > 290 {
					errorChan <- fmt.Errorf("error on page %d: %v", pageNum, err)
					return
				}

				var pageMsg map[string]any
				if err := json.Unmarshal(body, &pageMsg); err != nil {
					errorChan <- fmt.Errorf("json error on page %d: %v", pageNum, err)
					return
				}

				// Check if there are more pages
				if hasMore, ok := pageMsg["has_additional"].(bool); ok && !hasMore {
					hasAdditional = false
				}

				tags, ok := pageMsg["tags"].([]any)
				if !ok || len(tags) == 0 {
					return
				}

				pageTags := make([]string, 0, len(tags))
				for _, tag := range tags {
					if name, ok := tag.(map[string]any)["name"].(string); ok {
						pageTags = append(pageTags, name)
					}
				}
				// fmt.Println("page", pageNum)
				resultChan <- pageTags
			}(page)
			page++
		}
	}

	// Collect results in a separate goroutine
	var allTags []string
	done := make(chan bool)

	go func() {
		for tags := range resultChan {
			allTags = append(allTags, tags...)
		}
		done <- true
	}()

	// Wait for all requests to complete
	wg.Wait()
	close(resultChan)
	close(errorChan)
	<-done

	// Print results
	if len(allTags) == 0 {
		fmt.Println("No tags found or repository doesn't exist")
		return
	}

	fmt.Printf("\nFound %d tags:\n", len(allTags))
	for _, tag := range allTags {
		fmt.Println(tag)
	}
}

func (g Get) Find() {
	fmt.Println("Finding repo data for you ❤")

	// Buffered channels with larger capacity
	resultChan := make(chan []map[string]any, 100)
	errorChan := make(chan error, 100)

	// Increase concurrent workers significantly
	maxWorkers := 100
	sem := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup

	// Launch all pages immediately in parallel
	for page := 1; page <= 10; page++ { // Launch all 10 pages at once
		wg.Add(1)
		sem <- struct{}{} // Acquire semaphore

		go func(pageNum int) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore

			url := utils.GenUrl(fmt.Sprintf("find/repositories?page=%d&query=%s", pageNum, g.Repos))

			body, err, status := utils.Req(url, "GET", g.Token, nil)
			if err != nil || status > 290 {
				errorChan <- fmt.Errorf("error on page %d: %v", pageNum, err)
				return
			}

			var pageMsg map[string]any
			if err := json.Unmarshal(body, &pageMsg); err != nil {
				errorChan <- fmt.Errorf("json error on page %d: %v", pageNum, err)
				return
			}

			if results, ok := pageMsg["results"].([]any); ok {
				repos := make([]map[string]any, 0, len(results))
				for _, result := range results {
					if repo, ok := result.(map[string]any); ok {
						repos = append(repos, repo)
					}
				}
				resultChan <- repos
			}
		}(page)
	}

	// Collect results in a separate goroutine
	var allRepos []map[string]any
	done := make(chan bool)

	go func() {
		for repos := range resultChan {
			allRepos = append(allRepos, repos...)
		}
		done <- true
	}()

	// Wait for all requests to complete
	wg.Wait()
	close(resultChan)
	close(errorChan)
	<-done

	// Print results
	if len(allRepos) == 0 {
		fmt.Println("No repositories found")
		return
	}

	fmt.Printf("\nFound %d repositories:\n", len(allRepos))
	for _, repo := range allRepos {
		namespace := repo["namespace"].(map[string]any)
		fmt.Printf("%s/%s\n", namespace["name"], repo["name"])
	}
}

//seacrch repo
