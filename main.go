package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/gen2brain/dlgs"
)

const (
	apiURL   = "https://api.tonie.cloud/v2"
	tokenURL = "https://login.tonies.com/auth/realms/tonies/protocol/openid-connect/token"
)

func main() {
	conf, err := loadConfig()
	if err != nil {
		if !os.IsNotExist(err) {
			log.Fatalf("Error loading config: %v", err)
		}

		conf, err = configWizzard()
		if err != nil {
			log.Fatalf("Asking for config values: %v", err)
		}
	}

	dir, err := getDir()
	if err != nil {
		log.Fatalf("getting dir: %v", err)
	}

	if err := transferDir(dir, conf); err != nil {
		log.Fatalf("Error tranfering dir: %v", err)
	}

	fmt.Println("Upload compleate")
}

func getDir() (string, error) {
	if len(os.Args) > 1 {
		return os.Args[1], nil
	}

	path, ok, err := dlgs.File("Tonie upload", "", true)
	if err != nil {
		return "", fmt.Errorf("getting dir: %w", err)
	}
	if !ok {
		return "", fmt.Errorf("abort")
	}
	return path, nil
}

func transferDir(dir string, conf config) error {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("getting files: %v", err)
	}

	c, err := newConnection(conf)
	if err != nil {
		return fmt.Errorf("creating connection: %w", err)
	}

	var chapters []chapter
	for _, info := range files {
		if !info.Mode().IsRegular() {
			continue
		}

		path := dir + "/" + info.Name()

		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("opening file %s: %v", path, err)
		}

		fmt.Println("Processing:", info.Name())
		fileID, err := c.upload(f, int(info.Size()))
		if err != nil {
			return fmt.Errorf("uploading %s: %v", info.Name(), err)
		}

		chapters = append(chapters, chapter{Title: info.Name(), File: fileID})
	}

	if err := c.updateChapters(chapters); err != nil {
		return fmt.Errorf("updating chapters: %v", err)
	}
	return nil
}

type chapter struct {
	Title string `json:"title"`
	File  string `json:"file"`
}
