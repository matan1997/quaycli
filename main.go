package main

import (
	"flag"
	"fmt"
	"os"
	"quaycli/cmd"
	"quaycli/internal/utils"
	"strings"
)

func main() {
	command := os.Args[1]
	os.Args = append([]string{os.Args[0]}, os.Args[2:]...)

	flag.BoolVar(&utils.Force, "f", false, "force action no interface (short option)")
	organization := flag.String("organization", "", "organization")
	flag.StringVar(organization, "o", "", "organization (short option)")
	repo := flag.String("repo", "", "repo name")
	flag.StringVar(repo, "r", "", "repo (short option)")
	tag := flag.String("tag", "", "tag name")
	site := flag.String("site", "", "quay site")
	flag.StringVar(site, "s", "", "quay site (short option)")
	token := flag.String("token", "", "token")
	flag.StringVar(token, "t", "", "token (short option)")
	help := flag.Bool("help", false, "this helper")
	flag.BoolVar(help, "h", false, "this helper")

	flag.Parse()

	cfg := &utils.Config{
		Token:         *token,
		Organizations: *organization,
	}

	tokenList := strings.Split(*token, ",")
	tagsList := strings.Split(*tag, ",")

	if *help || len(os.Args) == 1 {
		fmt.Println(utils.HELPER)
		os.Exit(0)
	}

	if *token == "" {
		fmt.Printf("Must have token !\nSee 'quay --help'\n")
		os.Exit(0)
	}

	switch command {
	case "delete":
		deleteCmd := cmd.Delete{
			Config: cfg,
			Repos:  *repo,
			Tags:   *tag,
		}

		if utils.Force {
			fmt.Println(`Enabled force no user interface, I hope you know what are you doing..`)
		}

		if *tag == "" {
			deleteCmd.Repo()
		} else {
			deleteCmd.Tag()
		}
	case "get":
		getCmd := cmd.Get{
			Config: cfg,
			Repos:  *repo,
		}

		if *repo == "" {
			getCmd.Organization()
		} else {
			getCmd.Repo()
		}
	case "revert":
		postCmd := cmd.Post{
			Config: cfg,
			Repos:  *repo,
			Tags:   *tag,
		}

		if utils.Force {
			fmt.Println(`Enabled force no user interface, I hope you know what are you doing..`)
		}

		postCmd.RevertSha()
	case "mirror":
		postCmd := cmd.Post{
			Config: cfg,
		}

		if utils.Force {
			fmt.Println(`Enabled force no user interface, I hope you know what are you doing..`)
		}
		postCmd.MirrorRepo(tokenList, tagsList)
	case "find":
		getCmd := cmd.Get{
			Config: cfg,
			Repos:  *repo,
		}

		getCmd.Find()
	default:
		fmt.Printf("Unknown command: %s\nSee 'quay --help'\n", command)
	}
}
