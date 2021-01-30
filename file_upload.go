package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"

	"github.com/cheggaaa/pb/v3"
)

func uploadS3(content io.Reader, size int, meta io.Reader) (string, error) {
	var respBody struct {
		Request struct {
			URL    string            `json:"url"`
			Fields map[string]string `json:"fields"`
		} `json:"request"`
		FileID string `json:"fileId"`
	}

	if err := json.NewDecoder(meta).Decode(&respBody); err != nil {
		return "", fmt.Errorf("decoding meta data: %v", err)
	}

	buf := new(bytes.Buffer)
	multi := multipart.NewWriter(buf)

	for k, v := range respBody.Request.Fields {
		multi.WriteField(k, v)
	}

	fw, err := multi.CreateFormFile("file", respBody.Request.Fields["key"])
	if err != nil {
		return "", fmt.Errorf("creating part: %v", err)
	}

	if _, err := io.Copy(fw, content); err != nil {
		return "", fmt.Errorf("writing file: %v", err)
	}

	multi.Close()

	bar := pb.Full.Start(size)
	barReader := bar.NewProxyReader(buf)
	defer bar.Finish()

	req, err := http.NewRequest("POST", respBody.Request.URL, barReader)
	if err != nil {
		return "", fmt.Errorf("creating S3 request: %v", err)
	}
	req.ContentLength = int64(buf.Len())
	req.Header.Set("Content-Type", multi.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("sending file to S3: %v", err)
	}

	if resp.StatusCode > 299 {
		b, err := ioutil.ReadAll(resp.Body)
		body := string(b)
		if err != nil {
			body = fmt.Sprintf("can not read body: %v", err)
		}
		return "", fmt.Errorf("sendinf file to S3, got status %s: %s", resp.Status, body)
	}

	return respBody.FileID, nil
}
