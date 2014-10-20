package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"

	"github.com/codegangsta/negroni"
	"github.com/davecgh/go-spew/spew"
	apiClient "github.com/fsouza/go-dockerclient"
	"github.com/gorilla/mux"
	"github.com/unrolled/render"
)

type GithubPushEventPayload struct {
	Hook struct {
		Config struct {
			Secret string `json:"secret"`
		} `json:"config"`
	} `json:"hook"`
	Repository struct {
		FullName string `json:"full_name"`
		HtmlUrl  string `json:"html_url"`
	} `json:"repository"`
}

func main() {
	dockerHost := os.Getenv("DOCKER_HOST")
	if dockerHost == "" {
		dockerHost = "unix:///var/run/docker.sock"
	}

	_, err := apiClient.NewClient(dockerHost)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating client!!", err)
	}

	r := render.New(render.Options{})

	router := mux.NewRouter()
	router.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		var (
			payload GithubPushEventPayload
		)
		decoder := json.NewDecoder(req.Body)
		err := decoder.Decode(&payload)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error decoding Github push payload:", err)
		}
		spew.Dump(payload)
		repoPath := fmt.Sprintf("./repos/%s", payload.Repository.FullName)
		repoUrl := payload.Repository.HtmlUrl
		gitCmd := exec.Command("git")
		if _, err := os.Stat(repoPath); err != nil {
			if os.IsNotExist(err) {
				gitCmd = exec.Command("git", "clone", "--recursive", repoUrl, repoPath)
			} else {
				gitCmd = exec.Command(fmt.Sprintf("cd %s && git pull && cd -", repoPath))
			}
		}

		if err := gitCmd.Run(); err != nil {
			fmt.Fprintln(os.Stderr, "Error attempting git command on repo", payload.Repository.FullName, err)
		}
		r.JSON(w, http.StatusOK, "")
	}).Methods("POST")

	n := negroni.Classic()
	n.UseHandler(router)
	n.Run(":3000")
}
