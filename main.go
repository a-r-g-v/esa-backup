package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/upamune/go-esa/esa"
)

var esaClient *esa.Client

func main() {
	if err := realMain(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] %v\n", err)
		return
	}
}

func canAccessTeam(teamName string) (bool, error) {
	teams, err := esaClient.Team.GetTeams()
	if err != nil {
		return false, fmt.Errorf("GetTeams() failed")
	}

	for _, t := range teams.Teams {
		if t.Name == teamName {
			return true, nil
		}
	}
	return false, nil
}

func yieldAllPosts(teamName string, handle func(response esa.PostResponse)) error {
	page := "1"
	for {
		posts, err := esaClient.Post.GetPosts(teamName, url.Values{"per_page": {"100"}, "page": {page}})
		if err != nil {
			return fmt.Errorf("GetPosts failed: %w", err)
		}

		for _, post := range posts.Posts {
			handle(post)
		}
		if posts.NextPage == nil {
			return nil
		}
		page = strconv.Itoa(int(posts.NextPage.(float64)))
	}

}

func realMain(ctx context.Context) error {
	token := os.Getenv("ESA_ACCESS_TOKEN")
	if token == "" {
		return fmt.Errorf("os.Getenv(ESA_ACCESS_TOKEN) failed")
	}

	teamName := os.Getenv("ESA_TEAM_NAME")
	if teamName == "" {
		return fmt.Errorf("os.Getenv(ESA_TEAM_NAME) failed")
	}

	esaClient = esa.NewClient(token)

	canAccess, err := canAccessTeam(teamName)
	if err != nil {
		return fmt.Errorf("canAccessTeam failed: %w", err)
	}

	if !canAccess {
		return fmt.Errorf("you cannot access given teamName(%s)", teamName)
	}

	if err := yieldAllPosts(teamName, func(response esa.PostResponse) {
		backup(response)
	}); err != nil {
		return fmt.Errorf("yieldAllPosts failed: %w", err)
	}

	return nil
}

func backup(response esa.PostResponse) {
	response.FullName = "backup/" + response.FullName
	seps := strings.Split(response.FullName, "/")
	l := len(seps)

	path := strings.Join(seps[0:l-1], "/")

	if err := os.MkdirAll(path, 0o777); err != nil {
		if strings.Contains(err.Error(), "not a directory") {
			if err := os.Rename(path, path+"bk"); err != nil {
				panic(err)
			}

			if err := os.MkdirAll(path, 0o777); err != nil {
				panic(err)
			}

			if Exists(path + "/README") {
				panic(fmt.Sprintf("%s already exists", path+"/README"))
			}

			if err := os.Rename(path+"bk", path+"/README"); err != nil {
				panic(err)
			}
		} else {
			panic(err)
		}
	}

	if err := os.WriteFile(response.FullName, []byte(response.BodyMd), 0o666); err != nil {
		if strings.Contains(err.Error(), "is a directory") {
			if Exists(response.FullName + "/README") {
				panic(fmt.Sprintf("%s already exists", response.FullName+"/README"))
			}
			if err := os.WriteFile(response.FullName+"/README", []byte(response.BodyMd), 0o666); err != nil {
				panic(err)
			}

		} else {
			panic(fmt.Sprintf("os.WriteFile(%s). err: %v", response.FullName, err))
		}
	}

}

func Exists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}
