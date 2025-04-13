package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"quaycli/internal/utils"
	"sync"
	"time"
)

type Post struct {
	*utils.Config //get all multipule values
	Repos         string	
	Tags          string
}

type Robot struct {
	Name  string
	Token string
}

func makeMirrorRepo(region, organization, repo, token string) (int, error) {
	// Create repository post
	url := utils.GenUrl("repository")
	data := map[string]any{
		"repository":  repo,
		"visibility":  "private",
		"namespace":   organization,
		"description": "make repo with quay-cli for mirror",
		"repo_kind":   "image",
	}

	compactData, _ := json.Marshal(data)

	body, err, status := utils.Req(url, "POST", token, bytes.NewBuffer(compactData))

	if err != nil {
		return 0, err
	}

	if status > 290 {
		if status == 400 {
			fmt.Printf("Repo %s exist on %s\n", repo, region)
			return status, nil
		}
		return status, fmt.Errorf("status code: %d, error: %s", status, string(body))
	}

	fmt.Printf("Repo %s created on %s\n", repo, region)

	// Set repository state to mirror
	url = utils.GenUrl(fmt.Sprintf("repository/%s/%s/changestate", organization, repo))
	data = map[string]any{
		"state": "MIRROR",
	}

	compactData, _ = json.Marshal(data)

	body, err, status = utils.Req(url, "PUT", token, bytes.NewBuffer(compactData))

	if err != nil {
		return status, err
	}

	if status > 290 {
		return status, fmt.Errorf("status code: %d, error: %s", status, string(body))
	}

	return status, nil
}

func getRobot(organization, token string) (Robot, error) {
	url := utils.GenUrl(fmt.Sprintf("organization/%s/robots?permissions=true", organization))

	var msg map[string]any

	body, err, status := utils.Req(url, "GET", token, nil)

	if err != nil {
		fmt.Println(err)
		return Robot{}, err
	}

	err = json.Unmarshal(body, &msg)

	if err != nil {
		fmt.Println(err)
		return Robot{}, err
	}

	if status > 290 {
		fmt.Println("status:", status)
		fmt.Println(msg)
		return Robot{}, fmt.Errorf("status code greater than 290")
	}

	robots, ok := msg["robots"].([]any)
	if !ok {
		fmt.Println("invalid json format")
		return Robot{}, fmt.Errorf("invalid json format")
	}

	var maxRepos int
	var robotWithMostRepos Robot

	for _, r := range robots {
		robot, ok := r.(map[string]any)
		if !ok {
			continue
		}
		repos, ok := robot["repositories"].([]any)

		if !ok {
			continue
		}

		if len(repos) > maxRepos {
			maxRepos = len(repos)
			robotWithMostRepos = Robot{
				Name:  robot["name"].(string),
				Token: robot["token"].(string),
			}
		}
	}

	if maxRepos == 0 {
		fmt.Println("no robots found with repositories")
		return Robot{}, fmt.Errorf("no robots found with repositories")
	}

	return robotWithMostRepos, nil
}

// finished !!
func (p Post) MirrorRepo(token, tags []string) { //with token get robots and robot password

	m := utils.Caution{
		Message: "Do you want to mirror repo %s? (y/n)\n",
	}

	m.AskUser(p.Repos)

	fmt.Printf("Mirror repo %s for you ❤\n", p.Repos)

	targetRegion := "marganit"

	if p.Region == "" || p.Region == "metzuda" {
		p.Region = "metzuda"
	} else if p.Region == "marganit" {
		targetRegion = "metzuda"
	}

	//TODO: create repo and set as mirror
	makeRepoStatus, err := makeMirrorRepo(targetRegion, p.Organizations, p.Repos, token[1])
	if err != nil {
		fmt.Println(err)
		return
	}

	var (
		targetRobot, sourceRobot Robot
		targetErr, sourceErr     error
	)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		targetRobot, targetErr = getRobot(p.Organizations, token[1])
		if targetErr != nil {
			fmt.Println(targetErr)
		}
		wg.Done()
	}()

	go func() {
		sourceRobot, sourceErr = getRobot(p.Organizations, token[0])
		if sourceErr != nil {
			fmt.Println(sourceErr)
		}
		wg.Done()
	}()

	wg.Wait()

	url := utils.GenUrl(fmt.Sprintf("repository/%s/%s/mirror", p.Organizations, p.Repos))

	var msg map[string]any

	timeNow := time.Now().UTC()
	timeNow = timeNow.Truncate(time.Minute)

	data := map[string]any{
		"is_enabled":                 true,
		"external_reference":         "registry." + p.Region + "-1.idf.cts/" + p.Organizations + "/" + p.Repos,
		"sync_interval":              1800,                                           //30 minutes
		"sync_start_date":            string(timeNow.Format("2006-01-02T15:04:05Z")), //now
		"external_registry_username": sourceRobot.Name,
		"external_registry_password": sourceRobot.Token,
		"robot_username":             targetRobot.Name,
		"external_registry_config": map[string]any{
			"verify_tls":      false,
			"unsigned_images": false,
			"proxy": map[string]any{
				"http_proxy":  nil,
				"https_proxy": nil,
				"no_proxy":    nil,
			},
		},
		"root_rule": map[string]any{
			"rule_kind":  "tag_glob_csv",
			"rule_value": tags,
		},
	}

	compactData, _ := json.Marshal(data)

	method := "POST"

	if makeRepoStatus == 400 {
		method = "PUT"
	}

	body, err, status := utils.Req(url, method, token[1], bytes.NewBuffer(compactData))

	if err != nil {
		fmt.Println(err)
		return
	}

	_ = json.Unmarshal(body, &msg)

	if status > 290 {
		fmt.Println("status:", status)
		fmt.Println(msg)
		return
	}

	fmt.Printf("Repo %s is now mirror on %s 🚀\n", p.Repos, targetRegion)

}

// finised !!
func (p Post) RevertSha() {
	m := utils.Caution{
		Message: "Do you want to revert tag %s from repo %s on org %s? (y/n)\n",
	}

	m.AskUser(p.Tags, p.Repos, p.Organizations)

	fmt.Printf("Reverting tag %s for you ❤\n", p.Tags)

	if p.Region == "" {
		p.Region = "metzuda"
	}

	url := utils.GenUrl(fmt.Sprintf("repository/%s/%s/tag/?limit=1000&specificTag=%s", p.Organizations, p.Repos, p.Tags))

	//DEBUG
	// fmt.Println(url)
	// os.Exit(1)

	var msg map[string]any

	//get all tags sha
	body, err, status := utils.Req(url, "GET", p.Token, nil)

	if err != nil {
		fmt.Println(err)
		return
	}

	_ = json.Unmarshal(body, &msg)

	if status > 290 {
		fmt.Println("status:", status)
		fmt.Println(msg["error_message"].(string))
		return
	}

	if tags, ok := msg["tags"].([]any); !ok || len(tags) <= 1 {
		fmt.Println("there isnt more shas for this tag, cant revert.")
		return
	}

	shaDigest := msg["tags"].([]any)[1].(map[string]any)["manifest_digest"].(string)

	//revert to latest sha
	url = utils.GenUrl(fmt.Sprintf("repository/%s/%s/tag/%s/restore", p.Organizations, p.Repos, p.Tags))

	data := map[string]any{
		"manifest_digest": shaDigest,
	}

	compactData, _ := json.Marshal(data)

	body, err, status = utils.Req(url, "POST", p.Token, bytes.NewBuffer(compactData))

	if err != nil {
		fmt.Println(err)
		return
	}

	_ = json.Unmarshal(body, &msg)

	if status > 290 {
		fmt.Println("status:", status)
		fmt.Println(msg)
		return
	}
	
	fmt.Printf("tag %s reverted ! ♻\n", p.Tags)
}
