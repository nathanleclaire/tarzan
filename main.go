package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"

	"github.com/codegangsta/negroni"
	"github.com/davecgh/go-spew/spew"
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
	homeDir := os.Getenv("HOME")
	if _, err := os.Stat(fmt.Sprintf("%s/.dockercfg", homeDir)); err != nil {
		if os.IsNotExist(err) {
			cmd := exec.Command("docker", "login")
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := exec.Command("docker", "login").Run(); err != nil {
				fmt.Fprintln(os.Stderr, "Error running docker login")
				os.Exit(1)
			}
		}
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
		fullName := payload.Repository.FullName
		repoPath := fmt.Sprintf("./repos/%s", fullName)
		repoUrl := payload.Repository.HtmlUrl
		if _, err := os.Stat(repoPath); err != nil {
			if os.IsNotExist(err) {
				fmt.Println("Executing command", "git clone --recursive", repoUrl, repoPath)
				if err := exec.Command("git", "clone", "--recursive", repoUrl, repoPath).Run(); err != nil {
					fmt.Fprintln(os.Stderr, "Error cloning git repository:", err)
				}
			} else {
				fmt.Fprintln(os.Stderr, "Error stat-ing directory", repoPath, ":", err)
			}
		} else {
			os.Chdir(repoPath)
			fmt.Println("Pulling existing repository")
			if err := exec.Command("git", "pull").Run(); err != nil {
				fmt.Fprintln(os.Stderr, "Error pulling git repository:", err)
				r.JSON(w, http.StatusInternalServerError, "")
			}

			fmt.Println("Building docker image")
			// TODO: make it so that user name can be different than on Github
			if err := exec.Command("docker", "build", "-t", fullName, ".").Run(); err != nil {
				fmt.Fprintln(os.Stderr, "Error building docker image for", fullName, ":", err)
			}

			fmt.Println("Pushing image back to Docker Hub")
			if err := exec.Command("docker", "push", fullName).Run(); err != nil {
				fmt.Fprintln(os.Stderr, "Error pushing docker image for", fullName, ":", err)
			}
		}

		r.JSON(w, http.StatusOK, "")
	}).Methods("POST")

	n := negroni.Classic()
	n.UseHandler(router)
	n.Run(":80")
}
