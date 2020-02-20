package main

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type WorkflowRun struct {
	Id         int64  `json:"id"`
	Status     string `json:"status"`
	HeadSha    string `json:"head_sha"`
	HeadBranch string `json:"head_branch"`
}

type WorkflowRunsResponse struct {
	WorkflowRuns []WorkflowRun `json:"workflow_runs"`
}

var httpClient http.Client
var githubRepo = os.Getenv("GITHUB_REPOSITORY")
var githubToken = os.Getenv("GITHUB_TOKEN")
var branchName = strings.Replace(os.Getenv("GITHUB_REF"), "refs/heads/", "", 1)
var currentSha = os.Getenv("GITHUB_SHA")
var wg = sync.WaitGroup{}

func githubRequest(request *http.Request) (*http.Response, error) {
	request.Header.Set("Accept", "application/vnd.github.v3+json")
	request.Header.Set("Authorization", fmt.Sprintf("token %s", githubToken))
	response, err := httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// best effort
func cancelWorkflow(id int64) {
	request, err := http.NewRequest("POST", fmt.Sprintf(
		"https://api.github.com/repos/%s/actions/runs/%d/cancel", githubRepo, id), nil)
	if err != nil {
		log.Println(err)
	}
	response, err := githubRequest(request)
	if err != nil {
		log.Println(err)
	}
	if response.StatusCode != http.StatusAccepted {
		body, _ := ioutil.ReadAll(response.Body)
		log.Println(errors.New(fmt.Sprintf("failed to cancel workflow #%d, status code: %d, body: %s", id, response.StatusCode, body)))
	}
	wg.Done()
}

func main() {
	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	httpClient = http.Client{Transport: customTransport, Timeout: time.Minute}

	log.Printf("listing runs for branch %s in repo %s\n", branchName, githubRepo)
	request, err := http.NewRequest("GET", fmt.Sprintf("https://api.github.com/repos/%s/actions/runs", githubRepo), nil)
	if err != nil {
		panic(err)
	}
	query := request.URL.Query()
	query.Set("branch", branchName)
	request.URL.RawQuery = query.Encode()
	response, err := githubRequest(request)
	if err != nil {
		panic(err)
	}
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}
	var workflows WorkflowRunsResponse
	if err = json.Unmarshal(body, &workflows); err != nil {
		panic(err)
	}
	for _, run := range workflows.WorkflowRuns {
		if run.Status == "completed" {
			continue // not canceling completed jobs
		}
		if run.HeadBranch != branchName {
			continue // should not happen cuz we pre-filter, but better safe than sorry
		}
		if run.HeadSha == currentSha {
			continue // not canceling my own jobs
		}
		log.Printf("canceling run https://github.com/%s/actions/runs/%d\n", githubRepo, run.Id)
		wg.Add(1)
		go cancelWorkflow(run.Id)
	}
	wg.Wait()

}
