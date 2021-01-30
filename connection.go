package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"golang.org/x/oauth2"
)

type connection struct {
	conf   config
	client *http.Client
}

func newConnection(conf config) (*connection, error) {
	c := new(connection)
	c.conf = conf

	client, err := c.createClient()
	if err != nil {
		return nil, fmt.Errorf("creating client: %w", err)
	}
	c.client = client

	return c, nil
}

func (c *connection) createClient() (*http.Client, error) {
	conf := &oauth2.Config{
		ClientID: "meine-tonies",
		Endpoint: oauth2.Endpoint{
			TokenURL: tokenURL,
		},
	}
	token, err := conf.PasswordCredentialsToken(context.Background(), c.conf.Username, c.conf.Password)
	if err != nil {
		return nil, fmt.Errorf("getting token: %v", err)
	}

	return conf.Client(context.Background(), token), nil
}

func (c *connection) households() (map[string]string, error) {
	resp, err := c.client.Get(apiURL + "/households")
	if err != nil {
		return nil, fmt.Errorf("requesting households: %w", err)
	}
	if resp.StatusCode != 200 {
		b, err := ioutil.ReadAll(resp.Body)
		body := string(b)
		if err != nil {
			body = fmt.Sprintf("can not read body: %v", err)
		}
		return nil, fmt.Errorf("requesting households, got status %s: %s", resp.Status, body)
	}

	var households []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&households); err != nil {
		return nil, fmt.Errorf("decoding households: %w", err)
	}

	data := make(map[string]string, len(households))
	for _, h := range households {
		data[h.Name] = h.ID
	}
	return data, nil
}

func (c *connection) tonies(householdID string) (map[string]string, error) {
	resp, err := c.client.Get(fmt.Sprintf("%s/households/%s/creativetonies", apiURL, householdID))
	if err != nil {
		return nil, fmt.Errorf("requesting tonies: %w", err)
	}
	if resp.StatusCode != 200 {
		b, err := ioutil.ReadAll(resp.Body)
		body := string(b)
		if err != nil {
			body = fmt.Sprintf("can not read body: %v", err)
		}
		return nil, fmt.Errorf("requesting tonies, got status %s: %s", resp.Status, body)
	}

	var tonies []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tonies); err != nil {
		return nil, fmt.Errorf("decoding tonies: %w", err)
	}

	data := make(map[string]string, len(tonies))
	for _, h := range tonies {
		data[h.Name] = h.ID
	}
	return data, nil
}

func (c *connection) upload(r io.Reader, size int) (string, error) {
	resp, err := c.client.Post(apiURL+"/file", "application/json", nil)
	if err != nil {
		return "", fmt.Errorf("requesting file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, err := ioutil.ReadAll(resp.Body)
		body := string(b)
		if err != nil {
			body = fmt.Sprintf("can not read body: %v", err)
		}
		return "", fmt.Errorf("requesting file, got status %s: %s", resp.Status, body)
	}

	fileID, err := uploadS3(r, size, resp.Body)
	if err != nil {
		return "", fmt.Errorf("uploading file to S3: %v", err)
	}

	return fileID, nil
}

func (c *connection) updateChapters(chapters []chapter) error {
	creativeTonie := struct {
		Chapters []chapter `json:"chapters"`
	}{chapters}

	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(creativeTonie); err != nil {
		return fmt.Errorf("encoding tonie: %v", err)
	}

	req, err := http.NewRequest("PATCH", fmt.Sprintf("%s/households/%s/creativetonies/%s", apiURL, c.conf.HouseholdID, c.conf.TonieID), buf)
	if err != nil {
		return fmt.Errorf("creating put request: %v", err)
	}

	req.Header.Add("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("updating chappters: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, err := ioutil.ReadAll(resp.Body)
		body := string(b)
		if err != nil {
			body = fmt.Sprintf("can not read body: %v", err)
		}
		return fmt.Errorf("updating chapters, got status `%s`: %s", resp.Status, body)
	}
	return nil
}
