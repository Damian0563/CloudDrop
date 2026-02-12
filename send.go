package main

import (
	"archive/tar"
	"cloud.google.com/go/storage"
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/mdns"
	"github.com/schollz/progressbar/v3"
	"github.com/urfave/cli/v3"
	"io"
	"net"
	"os"
	"path/filepath"
)

func Drop(ctx context.Context, rootPath string, conn net.Conn, total int64) error {
	tw := tar.NewWriter(conn)
	defer tw.Close()
	bar := progressbar.DefaultBytes(total, "Sending")
	return filepath.WalkDir(rootPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, _ := d.Info()
		header, _ := tar.FileInfoHeader(info, "")
		relPath, _ := filepath.Rel(filepath.Dir(rootPath), path)
		header.Name = relPath
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if !d.IsDir() {
			f, _ := os.Open(path)
			io.Copy(tw, io.TeeReader(f, bar))
			f.Close()
		}
		return nil
	})
}

func getKey() string {

	return ""
}

func setKey(key string, url string) error {

	return nil
}

func sendPayload(filePath string, fileInfo os.FileInfo) (string, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return "", err
	}
	defer client.Close()
	bucket := client.Bucket("clouddrop")
	obj := bucket.Object(fileInfo.Name())
	w := obj.NewWriter(ctx)
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	bar := progressbar.DefaultBytes(fileInfo.Size(), "Sending")
	_, err = io.Copy(w, io.TeeReader(file, bar))
	if err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}
	url := fmt.Sprintf("gs://clouddrop/%s", fileInfo.Name())
	return url, nil
}

func superSend(ctx context.Context, c *cli.Command) error {
	if c.Args().Len() < 1 {
		return errors.New("you must provide a file path, no arguments provided")
	} else if c.Args().Len() > 1 {
		return errors.New("you can only provide one file path, too many arguments provided")
	}
	file := c.Args().First()
	fileInfo, err := os.Stat(file)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("filepath not resolved: %s", file)
		}
	}
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "./credentials.json")
	url, err := sendPayload(file, fileInfo)
	if err != nil {
		return err
	}
	key := getKey()
	err = setKey(key, url)
	return err
}

func Send(ctx context.Context, c *cli.Command) error {
	if c.Args().Len() < 1 {
		return errors.New("you must provide a file path, no arguments provided")
	} else if c.Args().Len() > 1 {
		return errors.New("you can only provide one file path, too many arguments provided")
	}
	file := c.Args().First()
	fileInfo, err := os.Stat(file)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", file)
		}
	}
	fmt.Println("Dropping file and waiting for recevier...")
	ln, _ := net.Listen("tcp", ":0")
	port := ln.Addr().(*net.TCPAddr).Port
	host, _ := os.Hostname()
	info := []string{"CloudDrop"}
	service, _ := mdns.NewMDNSService(host, "_clouddrop._tcp", "", "", port, nil, info)
	server, _ := mdns.NewServer(&mdns.Config{Zone: service})
	defer server.Shutdown()
	if conn, err := ln.Accept(); err != nil {
		return err
	} else {
		defer conn.Close()
		var total int64
		if fileInfo.IsDir() {
			files, _ := os.ReadDir(file)
			for _, f := range files {
				info, err := f.Info()
				if err != nil {
					continue
				}
				total += info.Size()
			}
		} else {
			total = fileInfo.Size()
		}
		return Drop(ctx, file, conn, total)
	}
}
