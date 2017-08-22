package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	git "gopkg.in/src-d/go-git.v4"
	cli "gopkg.in/urfave/cli.v1"
)

// A Resource contains data related to a local resource.
type Resource struct {
	Name     string
	Path     string
	Manifest *Manifest
}

func gatherResources(root string) []*Resource {
	resourceList := []*Resource{}

	filepath.Walk(root, func(p string, f os.FileInfo, err error) error {
		if err == nil {
			if f.IsDir() {
				manifest, err := openManifest(p)

				if err == nil {
					_, name := filepath.Split(p)

					resourceList = append(resourceList, &Resource{
						Name:     name,
						Path:     p,
						Manifest: manifest,
					})
				}
			}

			return nil
		}

		return err
	})

	return resourceList
}

func dirExists(dirName string) (bool, error) {
	stat, err := os.Stat(dirName)

	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}

	if !stat.IsDir() {
		return false, os.ErrExist
	}

	return true, nil
}

func cloneRepo(repoURI string, outRoot string) error {
	// clone the repository
	_, err := git.PlainClone(outRoot, false, &git.CloneOptions{
		URL: repoURI,
	})

	if err != nil {
		fmt.Printf("error while cloning %v: %v\n", repoURI, err)
		return err
	}

	return nil
}

func fetchRepo(outRoot string, force bool) error {
	repo, err := git.PlainOpen(outRoot)

	if err != nil {
		fmt.Printf("error while opening git repo %v: %v\n", outRoot, err)
		return err
	}

	worktree, err := repo.Worktree()

	if err != nil {
		fmt.Printf("error while opening git repo %v: %v\n", outRoot, err)
		return err
	}

	status, err := worktree.Status()

	if err != nil {
		fmt.Printf("error while getting git repo %v status: %v\n", outRoot, err)
		return err
	}

	if !status.IsClean() && !force {
		fmt.Printf("can't update %v, repo changes below:\n", outRoot)
		fmt.Printf("%v", status.String())
		fmt.Printf("use --force flag to bypass")

		return errors.New("Git repository is dirty, can't update")
	}

	err = worktree.Pull(&git.PullOptions{
		RemoteName: "origin",
	})

	if err != nil && err != git.NoErrAlreadyUpToDate {
		fmt.Printf("error while pulling git repo %v: %v\n", outRoot, err)
		return err
	}

	return nil
}

func addPackage(packageURI string, force bool) ([]*Resource, error) {
	repoURI := packageURI

	// split the package name part from the repo URI
	_, packageName := path.Split(repoURI)

	// split any . from the package name
	packageName = strings.Split(packageName, ".")[0]

	// format a root path of the form resources/[repoName]/
	packageRoot := fmt.Sprintf("[%v]", strings.ToLower(packageName))
	outRoot := filepath.Join("resources", packageRoot)

	// check for existence
	exists, err := dirExists(outRoot)

	if err != nil {
		return nil, err
	}

	// fetch/clone depending on existence
	if exists {
		err = fetchRepo(outRoot, force)
	} else {
		err = cloneRepo(repoURI, outRoot)
	}

	if err != nil {
		return nil, err
	}

	// gather resources in the repository
	resourceList := gatherResources(outRoot)

	// gather dependencies
	dependencies := map[string]bool{}

	for _, resource := range resourceList {
		deps := resource.Manifest.GetAll("dependency_url")

		for _, dep := range deps {
			dependencies[dep] = true
		}
	}

	// fetch dependencies
	for dep := range dependencies {
		localList, err := addPackage(dep, force)

		if err != nil {
			return nil, err
		}

		resourceList = append(localList, resourceList...)
	}

	return resourceList, nil
}

func parseResourceLines(file string) ([]string, error) {
	fileData, err := ioutil.ReadFile(file)

	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	// if not existent, assume an empty file
	if err != nil {
		fileData = []byte{}
	}

	fileText := string(fileData)

	lines := strings.Split(fileText, "\n")

	// read any 'start' lines
	resources := []string{}

	for _, line := range lines {
		if strings.HasPrefix(line, "start ") {
			splitLine := strings.Split(line, " ")

			if len(splitLine) >= 2 {
				resources = append(resources, strings.TrimSpace(splitLine[1]))
			}
		}
	}

	return resources, nil
}

func writeResourceLines(file string, lines []string) error {
	var buffer bytes.Buffer
	writer := bufio.NewWriter(&buffer)

	fmt.Fprintf(writer, "# THIS FILE IS AUTOMATICALLY-GENERATED\r\n")
	fmt.Fprintf(writer, "# DO NOT ADD ANYTHING OTHER THAN `start` LINES\r\n")

	for _, line := range lines {
		fmt.Fprintf(writer, "start %v\r\n", line)
	}

	writer.Flush()

	err := ioutil.WriteFile(file, buffer.Bytes(), 0644)

	return err
}

func addResources(resources []*Resource, file string) error {
	// add lines to resource configuration
	resourceLines, err := parseResourceLines(file)

	if err != nil {
		return err
	}

	// try finding/adding each resource
	for _, resource := range resources {
		// loop through each line to see if we found any
		found := false

		for _, line := range resourceLines {
			if line == resource.Name {
				found = true
				break
			}
		}

		if !found {
			resourceLines = append(resourceLines, resource.Name)
		}
	}

	err = writeResourceLines(file, resourceLines)

	return err
}

func cmdAdd(ctx *cli.Context) error {
	// get the intended package URI
	packageURI := ctx.Args().First()

	if len(strings.TrimSpace(packageURI)) == 0 {
		return nil
	}

	// fetch the package
	resources, err := addPackage(packageURI, ctx.Bool("force"))

	if err != nil {
		return err
	}

	err = addResources(resources, ctx.String("config-file"))

	if err != nil {
		return err
	}

	return nil
}

func cmdGet(ctx *cli.Context) error {
	// get the intended package URI
	packageURI := ctx.Args().First()

	if len(strings.TrimSpace(packageURI)) == 0 {
		return nil
	}

	// fetch the package
	resources, err := addPackage(packageURI, ctx.Bool("force"))

	for _, resource := range resources {
		fmt.Printf("- %s\n", resource.Name)
	}

	return err
}
