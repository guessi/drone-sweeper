package main

import (
	"os"
	"strconv"

	"github.com/drone/drone-go/drone"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	host := os.Getenv("DRONE_HOST")
	token := os.Getenv("DRONE_TOKEN")
	namespace := os.Getenv("DRONE_NAMESPACE")
	repo := os.Getenv("DRONE_REPO_NAME")

	page, _ := strconv.Atoi(os.Getenv("DRONE_REPO_PAGE"))
	size, _ := strconv.Atoi(os.Getenv("DRONE_REPO_PAGE_SIZE"))
	before, _ := strconv.Atoi(os.Getenv("DRONE_PURGE_BEFORE"))
	purgeBuilds, _ := strconv.ParseBool(os.Getenv("DRONE_PURGE_BUILDS"))
	purgeLogs, _ := strconv.ParseBool(os.Getenv("DRONE_PURGE_LOGS"))

	c := newClient(host, token)
	bs := getBuilds(c, namespace, repo, page, size)

	if purgeLogs {
		cleanupLogs(c, bs, namespace, repo, before)
	} else {
		log.Infof("DRONE_PURGE_LOGS=false detected, all logs untouch")
	}

	if purgeBuilds {
		cleanupBuilds(c, namespace, repo, before)
	} else {
		log.Infof("DRONE_PURGE_BUILDS=false detected, all builds untouch")
	}
}

func newClient(host, token string) drone.Client {
	config := new(oauth2.Config)
	auther := config.Client(
		oauth2.NoContext,
		&oauth2.Token{
			AccessToken: token,
		},
	)
	return drone.NewClient(host, auther)
}

func getBuilds(c drone.Client, ns, repo string, page, pageSize int) []*drone.Build {
	builds, err := c.BuildList(ns, repo, drone.ListOptions{
		Page: page,
		Size: pageSize,
	})
	if err != nil {
		log.Fatalf("Oops... failed retrieving build list\n")
	}
	return builds
}

func cleanupLogs(c drone.Client, bs []*drone.Build, ns, repo string, before int) {
	for _, b := range bs {
		if int(b.Number) >= before {
			continue
		}

		log.Infof("[Build %d] - %s (%s)\n", int(b.Number), b.Message, b.AuthorName)
		buildInfo, err := c.Build(ns, repo, int(b.Number))
		if err != nil {
			log.Fatalf("[debug] Failed to get build info\n")
		}

		for _, stage := range buildInfo.Stages {
			log.Infof("%-4s[Stage %d] - %s\n", "", stage.Number, stage.Name)
			for _, step := range stage.Steps {
				log.Infof("%-8s[Step %d] - %s\n", "", step.Number, step.Name)

				log.Infof("%-12s[debug] purging log", "")
				if err := c.LogsPurge(ns, repo, int(b.Number), stage.Number, step.Number); err != nil {
					log.Fatalf("%-12s[debug] log purge failed", "")
				}
			}
		}
	}
}

func cleanupBuilds(c drone.Client, ns, repo string, before int) {
	log.Infof("%-4s[debug] purging build", "")
	if err := c.BuildPurge(ns, repo, before); err != nil {
		log.Fatalf("%-4s[debug] build purge failed", "")
	}
}
