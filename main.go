package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"

	"github.com/codegangsta/negroni"
	"github.com/davecgh/go-spew/spew"
	"github.com/gorilla/mux"
	logging "github.com/op/go-logging"
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

var log = logging.MustGetLogger("streamLog")
var format = "%{color}%{time:15:04:05} ▶ %{color:reset} %{message}"

func streamCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(cmdReader)
	go func() {
		for scanner.Scan() {
			log.Notice(scanner.Text())
		}
	}()
	err = cmd.Start()
	if err != nil {
		return err
	}
	err = cmd.Wait()
	if err != nil {
		return err
	}
	return nil
}

func main() {
	logBackend := logging.NewLogBackend(os.Stderr, "", 0)
	logging.SetBackend(logBackend)
	logging.SetFormatter(logging.MustStringFormatter(format))
	logging.SetLevel(logging.NOTICE, "streamLog")
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
			err := streamCommand("docker", "build", "-t", fullName, ".")
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error building docker image for", fullName, ":", err)
				r.JSON(w, http.StatusInternalServerError, map[string]interface{}{
					"Error": err,
				})
			}

			fmt.Println("Pushing image back to Docker Hub")
			err = streamCommand("docker", "push", fullName)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error pushing docker image for", fullName, ":", err)
				r.JSON(w, http.StatusInternalServerError, map[string]interface{}{
					"Error": err,
				})
			}
		}

		r.JSON(w, http.StatusOK, "")
	}).Methods("POST")

	n := negroni.Classic()
	n.UseHandler(router)
	n.Run(":80")
}
