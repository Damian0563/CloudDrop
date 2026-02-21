package main

import (
	"archive/tar"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hashicorp/mdns"
	"github.com/schollz/progressbar/v3"
	"github.com/urfave/cli/v3"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func handleStream(conn net.Conn) error {
	tr := tar.NewReader(conn)
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
			os.MkdirAll(header.Name, 0755)
		case tar.TypeReg:
			fmt.Printf("Receiving file: %s\n", header.Name)
			os.MkdirAll(filepath.Dir(header.Name), 0755)
			outFile, _ := os.Create(header.Name)
			bar := progressbar.DefaultBytes(header.Size, "Downloading "+header.Name)
			io.Copy(outFile, io.TeeReader(tr, bar))
			outFile.Close()
		}
	}
	return nil
}

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
	resp, err := http.Get(url)
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
		bar := progressbar.DefaultBytes(resp.ContentLength, "Downloading"+originalName)
		_, err = io.Copy(outFile, io.TeeReader(resp.Body, bar))
	} else {
		bar := progressbar.DefaultBytes(-1, "Downloading")
		_, err = io.Copy(outFile, io.TeeReader(resp.Body, bar))
	}
	return err
}

type receiveResponse struct {
	Status       string `json:"status"`
	Error        string `json:"error"`
	Msg          string `json:"msg"`
	OriginalName string `json:"original_name"`
	IsDir        bool   `json:"is_dir"`
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
	resp, err := http.DefaultClient.Do(req)
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

func Receive(ctx context.Context, c *cli.Command) error {
	log.SetOutput(io.Discard)
	fmt.Println("Awaiting for peers...")
	entriesCh := make(chan *mdns.ServiceEntry, 4)
	done := make(chan bool)
	errorCh := make(chan error)
	go func() {
		for entry := range entriesCh {
			target := fmt.Sprintf("%s:%d", entry.AddrV4, entry.Port)
			conn, err := net.Dial("tcp", target)
			if err != nil {
				errorCh <- err
				return
			}
			err = handleStream(conn)
			conn.Close()
			if err != nil {
				errorCh <- err
			} else {
				done <- true
				return
			}
		}
	}()
	params := mdns.DefaultParams("_clouddrop._tcp")
	params.Entries = entriesCh
	params.WantUnicastResponse = true
	params.DisableIPv6 = true
	err := mdns.Query(params)
	if err != nil {
		return err
	}
	<-done
	return nil
}
