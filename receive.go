package main

import (
	"archive/tar"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/schollz/progressbar/v3"
	"github.com/urfave/cli/v3"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func extractTar(r io.Reader) error {
	tr := tar.NewReader(r)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		switch header.Typeflag {
		case tar.TypeDir:
			fmt.Printf("Creating directory: %s\n", header.Name)
			if err := os.MkdirAll(header.Name, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(header.Name), 0755); err != nil {
				return err
			}
			outFile, err := os.Create(header.Name)
			if err != nil {
				return err
			}
			_, err = io.Copy(outFile, tr)
			outFile.Close()
			if err != nil {
				return err
			}
			fmt.Printf("  ✓ %s\n", header.Name)
		}
	}
	return nil
}

func downloadUrl(url string, originalName string, isDir bool) error {
	resp, err := insecureClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if isDir {
		fmt.Printf("Receiving directory: %s\n", originalName)
		if err := os.MkdirAll(originalName, 0755); err != nil {
			return err
		}
		if err := os.Chdir(originalName); err != nil {
			return err
		}
		defer os.Chdir("..")
		return extractTar(resp.Body)
	}

	filename := filepath.Base(url)
	if strings.Contains(filename, "?") {
		filename = strings.Split(filename, "?")[0]
	}
	if filename == "" || filename == "/" {
		filename = "download"
	}
	outFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer outFile.Close()

	if resp.ContentLength > 0 {
		bar := progressbar.DefaultBytes(resp.ContentLength, "Downloading "+originalName)
		_, err = io.Copy(outFile, io.TeeReader(resp.Body, bar))
	} else {
		bar := progressbar.DefaultBytes(-1, "Downloading")
		_, err = io.Copy(outFile, io.TeeReader(resp.Body, bar))
	}
	return err
}

func checkTimeout() error {
	thisTimestamp := time.Now().Unix()
	file, err := os.OpenFile("timeout.txt", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer file.Close()
	content, err := io.ReadAll(file)
	if err != nil {
		return err
	}
	if len(content) == 0 {
		_, err = file.WriteAt([]byte(fmt.Sprintf("%d\n", thisTimestamp)), 0)
		return err
	}
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	lastTimeStamp, err := strconv.ParseInt(lines[0], 10, 64)
	if err != nil {
		return err
	}
	secondLast := int64(0)
	if len(lines) > 1 {
		secondLast, _ = strconv.ParseInt(lines[1], 10, 64)
	}
	timeout := thisTimestamp - lastTimeStamp
	if secondLast > 0 {
		timeout = thisTimestamp - secondLast
	}
	if timeout < 10 {
		return fmt.Errorf("please wait %d seconds", 10-timeout)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if err := file.Truncate(0); err != nil {
		return err
	}
	if secondLast == 0 {
		_, err = file.WriteAt([]byte(fmt.Sprintf("%d\n%d\n", lastTimeStamp, thisTimestamp)), 0)
		return err
	}
	_, err = file.WriteAt([]byte(fmt.Sprintf("%d\n%d\n", secondLast, thisTimestamp)), 0)
	return err
}

func superReceive(ctx context.Context, c *cli.Command) error {
	if c.Args().Len() < 1 {
		return errors.New("you must provide a code, no arguments provided")
	} else if c.Args().Len() > 1 {
		return errors.New("you can only provide one code, too many arguments provided")
	}
	if err := checkTimeout(); err != nil {
		return err
	}
	code := c.Args().First()
	authority := os.Getenv("AUTHORITY")
	if authority == "" {
		authority = defaultAuthority
	}
	authority = authority + "/receive/" + code
	req, err := http.NewRequest("GET", authority, nil)
	secret := os.Getenv("SECRET")
	if secret == "" {
		secret = defaultSecret
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", secret))
	if err != nil {
		return err
	}
	resp, err := insecureClient.Do(req)
	if err != nil {
		return err
	}
	resBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	Response := receiveResponse{}
	err = json.Unmarshal(resBody, &Response)
	if err != nil {
		return err
	}
	if Response.Status != "ok" {
		return errors.New(Response.Error)
	}
	err = downloadUrl(Response.Msg, Response.OriginalName, Response.IsDir)
	if err != nil {
		return err
	}
	return nil
}
