package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/codegangsta/cli"
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
var format = "%{color}%{time:15:04:05} => %{color:reset} %{message}"

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

func BuildHookReceiver(c *cli.Context, r *render.Render, dockerBinary string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		var (
			payload GithubPushEventPayload
		)
		decoder := json.NewDecoder(req.Body)
		err := decoder.Decode(&payload)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error decoding Github push payload:", err)
		}
		spew.Dump(payload)
		if c.String("secret") == "" || payload.Hook.Config.Secret == c.String("secret") {
			githubFullName := payload.Repository.FullName
			repoPath := fmt.Sprintf("./repos/%s", githubFullName)
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
				os.Chdir(repoPath)
			} else {
				os.Chdir(repoPath)
				fmt.Println("Pulling existing repository")
				if err := exec.Command("git", "pull").Run(); err != nil {
					fmt.Fprintln(os.Stderr, "Error pulling git repository:", err)
					r.JSON(w, http.StatusInternalServerError, "")
				}
			}

			namespacedImage := ""
			splitImage := strings.Split(githubFullName, "/")
			imageBase := splitImage[len(splitImage)-1]
			if c.String("hub-name") == "" {
				if c.String("alt-registry") == "" {
					namespacedImage = githubFullName
				} else {
					namespacedImage = fmt.Sprintf("%s/%s", c.String("alt-registry"), imageBase)
				}
			} else {
				namespacedImage = fmt.Sprintf("%s/%s", c.String("hub-name"), imageBase)
			}

			fmt.Println("Building docker image")
			err := streamCommand(dockerBinary, "build", "-t", namespacedImage, ".")
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error building docker image for", namespacedImage, ":", err)
				r.JSON(w, http.StatusInternalServerError, map[string]interface{}{
					"Error": err,
				})
			}

			registryName := ""
			if c.String("alt-registry") != "" {
				registryName = c.String("alt-registry")
			} else {
				registryName = "Docker Hub"
			}

			fmt.Println(fmt.Sprintf("Pushing image back to specified registry (%s)...", registryName))
			err = streamCommand(dockerBinary, "push", namespacedImage)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error pushing docker image for", namespacedImage, ":", err)
				r.JSON(w, http.StatusInternalServerError, map[string]interface{}{
					"Error": err,
				})
			}
		} else {
			r.JSON(w, http.StatusInternalServerError, map[string]interface{}{
				"Error": "Secret from payload was invalid",
			})
		}
		r.JSON(w, http.StatusOK, "")
	}
}

func main() {
	logBackend := logging.NewLogBackend(os.Stderr, "", 0)
	logging.SetBackend(logBackend)
	logging.SetFormatter(logging.MustStringFormatter(format))
	logging.SetLevel(logging.NOTICE, "streamLog")
	homeDir := os.Getenv("HOME")

	app := cli.NewApp()
	app.Name = "tarzan"
	app.Usage = "naive cached automated build implementation"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "p,port",
			Value: "3000",
			Usage: "port to serve tarzan on",
		},
		cli.StringFlag{
			Name:  "alt-registry",
			Value: "",
			Usage: "alternative registry to push images to instead of Docker Hub",
		},
		cli.StringFlag{
			Name:  "secret",
			Value: "",
			Usage: "secret to use when receiving webhook payload",
		},
		cli.StringFlag{
			Name:  "hub-name",
			Value: "",
			Usage: "specify a username on Docker Hub which is different than your Github handle",
		},
		cli.StringFlag{
			Name:  "docker-binary-name",
			Value: "docker",
			Usage: "specify the docker binary name (if it is not docker in $PATH)",
		},
	}

	app.Action = func(c *cli.Context) {
		dockerBinary := c.String("docker-binary-name")
		if _, err := os.Stat(fmt.Sprintf("%s/.dockercfg", homeDir)); err != nil {
			if os.IsNotExist(err) {
				log.Warning("Detected no Docker Hub login.  Please log in now.")
				cmd := exec.Command(dockerBinary, "login")
				cmd.Stdin = os.Stdin
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				if err := cmd.Run(); err != nil {
					fmt.Fprintln(os.Stderr, "Error running docker login")
					os.Exit(1)
				}
			}
		}
		r := render.New(render.Options{})
		router := mux.NewRouter()
		router.HandleFunc("/build", BuildHookReceiver(c, r, dockerBinary)).Methods("POST")

		n := negroni.Classic()
		n.UseHandler(router)
		n.Run(fmt.Sprintf(":%s", c.String("port")))
	}

	app.Run(os.Args)
}
