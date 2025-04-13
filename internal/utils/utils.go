package utils

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
)

var Force bool

const (
	API_PATH = "api/v1"
	HELPER   = `Hi There !
This CLI for quay
SYNTAX:
quay [COMMAND] [FLAGS] 

COMMANDS:
delete
get
mirror
revert

FLAGS:
-o, --organization: Wanted organization
-t, --token: Token that have permission
-s, --site: Wanted site (default is metzuda)
-r, --repo: Wanted repo
-f, --force: Enabled force no user interface
--tag: tag name

EX:
quay delete -o <your organiztaion> --token <token> -s <wanted site> -r <repo name>
quay delete -o <your organiztaion> --token <token> -s <wanted site> -r <repo name> --tag <tag name>
quay get -o <your organiztaion> --token <token> -s <wanted site> -r <repo name>
quay revert -o <your organiztaion> --token <token> -r <repo name> --tag <tag name> -f
quay mirror -o <your organiztaion> -t <source token>,<target token> -s <source site> -r <repo name> --tags tag1,tag2

LIFE-HACK:
Be good be opensource :)`
)

type Config struct {
	Region        string
	Token         string
	Organizations string
}

type Caution struct {
	Message string
}

// askUser is function that ask user to confirm his action, it takes many strings paramater and message var
func (c Caution) AskUser(a ...any) {
	if !Force {
		fmt.Printf(c.Message, a...)
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))
		if input != "y" && (input != "yes") {
			os.Exit(0)
		}
	}
}

// genUrl is function that genrate the needed API url with region
func GenUrl(path string) string {
	return fmt.Sprintf("https://quay.io/%s/%s", API_PATH, path)
}

// preReq is function that prepare request before sending it
func preReq(req *http.Request, token string) *http.Client {
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")

	// Disable ssl verify
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}

	return client
}

// req is function that send request and return the json, error and status as int
func Req(url, method, token string, data io.Reader) ([]byte, error, int) {

	statusCode := 500

	req, err := http.NewRequest(method, url, data)
	if err != nil {
		return nil, err, statusCode
	}

	client := preReq(req, token)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err, statusCode
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("get status error", err)
		return nil, err, statusCode
	}

	statusStr := strings.Fields(resp.Status)[0]
	statusCode, err = strconv.Atoi(statusStr)
	if err != nil {
		fmt.Println("get status error", err)
		return nil, err, statusCode
	}

	return body, nil, statusCode
}
