package main

import(
	"log"
	"fmt"
	"os"
	"os/exec"
	"bufio"
)

func releaseRepository(payload GitHubPushEventPayload) {
	log.Println(payload)
	log.Println("Entering Build stage")
	err := dockerBuild(payload)
	if (err != nil){
		log.Println("Build failed.")
		return
	}
	log.Println("Entering Push stage")
	err = dockerPush(payload)
	if (err != nil){
		log.Println("Push failed.")
		return
	}
	log.Printf("Tags should be available latest & %s", payload.Release.Tag_Name)
}
func dockerBuild(payload GitHubPushEventPayload) (error){
	imageNameTag := fmt.Sprintf("%s/%s:%s", os.Getenv("DOCKERHUB_ORG"), payload.Repository.Name, payload.Release.Tag_Name)
	imageNameLatest := fmt.Sprintf("%s/%s:latest", os.Getenv("DOCKERHUB_ORG"), payload.Repository.Name)
	log.Printf("Building image %s and %s\n", imageNameTag, imageNameLatest)

	err := streamCommand(true, "docker", "build", "--force-rm", "--pull", "--tag", imageNameTag, "--tag", imageNameLatest, fmt.Sprintf("%s#%s",payload.Repository.CloneUrl, payload.Release.Tag_Name))
	if (err != nil){
		log.Println(err)
	}
	return nil
}
func dockerPush(payload GitHubPushEventPayload) (error){
	imageNameTag := fmt.Sprintf("%s/%s:%s", os.Getenv("DOCKERHUB_ORG"), payload.Repository.Name, payload.Release.Tag_Name)
	imageNameLatest := fmt.Sprintf("%s/%s:latest", os.Getenv("DOCKERHUB_ORG"), payload.Repository.Name)
	
	log.Printf("Pushing image %s and %s\n", imageNameTag, imageNameLatest)
	err := streamCommand(false, "docker", "login", "--username", os.Getenv("DOCKERHUB_NAME"), "--password", os.Getenv("DOCKERHUB_PW"))
	if (err != nil) {
		log.Printf("Error at docker login with user %s", os.Getenv("DOCKERHUB_NAME"))
		log.Println(err)
		return err
	}
	err = streamCommand(true, "docker", "push", imageNameTag)
	if err != nil {
		log.Printf("Error pushing docker image for %s: %s", imageNameTag, err)
		return err
	}
	err = streamCommand(true, "docker", "push", imageNameLatest)
	if err != nil {
		log.Printf("Error pushing docker image for %s: %s", imageNameLatest, err)
		return err
	}
	return nil
}

func streamCommand(logNoSecret bool, name string, args ...string) error {
	if (os.Getenv("LOG_STREAMCMD") == "true" && logNoSecret){
		log.Printf("Issuing command %s with params:" , name)
		log.Println(args)
	}
	cmd := exec.Command(name, args...)
	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(cmdReader)
	go func() {
		for scanner.Scan() {
			if (os.Getenv("LOG_STREAMCMD") == "true") {
				log.Println(scanner.Text())
			}
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

