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
	"strings"
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
		bar := progressbar.DefaultBytes(resp.ContentLength, "Downloading")
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

func superReceive(ctx context.Context, c *cli.Command) error {
	if c.Args().Len() < 1 {
		return errors.New("you must provide a code, no arguments provided")
	} else if c.Args().Len() > 1 {
		return errors.New("you can only provide one code, too many arguments provided")
	}
	code := c.Args().First()
	authority := os.Getenv("AUTHORITY") + "/receive/" + code
	req, err := http.NewRequest("GET", authority, nil)
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
	fmt.Println("Receiving files...")
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
