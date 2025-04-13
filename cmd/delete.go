package cmd

import (
	"encoding/json"
	"fmt"
	"quaycli/internal/utils"
)

type Delete struct {
	*utils.Config //get all multipule values
	Repos         string
	Tags          string
}

// finished
func (d Delete) Repo() {
	m := utils.Caution{
		Message: "Do you want to delete repo %s from org %s? (y/n)\n",
	}

	m.AskUser(d.Repos, d.Organizations)

	fmt.Printf("Deleting repo %s for you ❤\n", d.Repos)

	url := utils.GenUrl(fmt.Sprintf("repository/%s/%s", d.Organizations, d.Repos))

	var msg map[string]any

	body, err, status := utils.Req(url, "DELETE", d.Token, nil)

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

	fmt.Printf("Repo %s deleted ! ❌\n", d.Repos)
}

// finished !!
func (d Delete) Tag() {
	m := utils.Caution{
		Message: "Do you want to delete tag %s from repo %s on org %s? (y/n)\n",
	}

	m.AskUser(d.Tags, d.Repos, d.Organizations)

	fmt.Printf("Deleting tag %s for you ❤\n", d.Tags)

	url := utils.GenUrl(fmt.Sprintf("repository/%s/%s/tag/%s", d.Organizations, d.Repos, d.Tags))

	var msg map[string]any

	body, err, status := utils.Req(url, "DELETE", d.Token, nil)

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

	fmt.Printf("tag %s deleted ! ❌\n", d.Tags)
}
