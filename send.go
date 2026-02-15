package main

import (
	"archive/tar"
	"cloud.google.com/go/storage"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hashicorp/mdns"
	"github.com/schollz/progressbar/v3"
	"github.com/urfave/cli/v3"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
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
	adjectives := []string{"Jumping", "Flying", "Speeding", "Fast", "Dirty", "Playful", "Cloudy", "Hot", "Blessed", "Clean", "Swift", "Cold", "Rising", "Sad", "Happy", "Exotic", "Sunny", "Poisoned", "Sweet", "Great", "Skilled", "Wise", "Smart", "Safe", "Euphoric", "Classy", "Feaverish"}
	nouns := []string{"Jack", "Wasp", "Dragon", "Panther", "Cat", "Fox", "Dog", "Ghost", "Lion", "Peter", "Hat", "Flee", "Ant", "Watch", "Cloud", "Sun", "Moon", "Star", "Rock", "Max", "Beaver", "Feather", "Plum", "Cherry", "Brush", "Berry", "Master", "Student", "Player", "Sam", "Arnold", "Ring", "Thief", "Judge"}
	numbers := []string{}
	for i := range 100 {
		numbers = append(numbers, strconv.Itoa(i))
	}
	key := adjectives[rand.Intn(len(adjectives)-1)] + nouns[rand.Intn(len(nouns)-1)] + numbers[rand.Intn(len(numbers)-1)]
	return key
}

func generateSignedUrl(gs_url string) (string, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return "", err
	}
	defer client.Close()
	filename := strings.Split(gs_url, "/")[3]
	opts := &storage.SignedURLOptions{
		Scheme:  storage.SigningSchemeV4,
		Method:  "GET",
		Expires: time.Now().Add(16 * time.Minute),
	}
	u, err := client.Bucket("clouddrop").SignedURL(filename, opts)
	if err != nil {
		return "", err
	}
	return u, nil
}

func sendPayload(filePath string, fileInfo os.FileInfo) (string, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return "", err
	}
	defer client.Close()
	bucketName := os.Getenv("BUCKET_NAME")
	bucket := client.Bucket(bucketName)
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
	url := fmt.Sprintf("gs://%s/%s", bucketName, fileInfo.Name())
	return url, nil
}

type Response struct {
	Ok    string `json:"ok"`
	Error string `json:"error"`
}

func setKey(key string, url string) error {
	authority := os.Getenv("AUTHORITY") + "/drop"
	req, err := http.NewRequest("POST", authority, strings.NewReader(fmt.Sprintf(`{"key":"%s","url":"%s"}`, key, url)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	resBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	fmt.Println(string(resBody))
	Response := Response{}
	err = json.Unmarshal(resBody, &Response)
	if err != nil {
		return err
	}
	if Response.Ok != "ok" {
		return errors.New(Response.Error)
	}
	return nil
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
	credentials := os.Getenv("CREDENTIALS_PATH")
	err = os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", credentials)
	if err != nil {
		return err
	}
	gsUrl, err := sendPayload(file, fileInfo)
	if err != nil {
		return err
	}
	key := getKey()
	signedUrl, err := generateSignedUrl(gsUrl)
	if err != nil {
		return err
	}
	if err = setKey(key, signedUrl); err != nil {
		return err
	}
	fmt.Println("Access key:", key)
	return nil
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
