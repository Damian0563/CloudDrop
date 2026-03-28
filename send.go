package main

import (
	"archive/tar"
	"cloud.google.com/go/storage"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/schollz/progressbar/v3"
	"github.com/urfave/cli/v3"
	"google.golang.org/api/option"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

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

func createTarArchive(sourcePath string) (string, int64, error) {
	tarFile, err := os.CreateTemp("", "clouddrop-*.tar")
	if err != nil {
		return "", 0, err
	}
	tw := tar.NewWriter(tarFile)
	err = filepath.WalkDir(sourcePath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(filepath.Dir(sourcePath), path)
		if err != nil {
			return err
		}
		header.Name = relPath
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if !d.IsDir() {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			_, err = io.Copy(tw, f)
			f.Close()
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		tw.Close()
		tarFile.Close()
		os.Remove(tarFile.Name())
		return "", 0, err
	}

	if err := tw.Close(); err != nil {
		tarFile.Close()
		os.Remove(tarFile.Name())
		return "", 0, err
	}
	if err := tarFile.Close(); err != nil {
		os.Remove(tarFile.Name())
		return "", 0, err
	}
	stat, err := os.Stat(tarFile.Name())
	if err != nil {
		os.Remove(tarFile.Name())
		return "", 0, err
	}

	return tarFile.Name(), stat.Size(), nil
}

func generateSignedUrl(gsUrl string, originalName string) (string, error) {
	ctx := context.Background()
	googleCredentials := os.Getenv("GOOGLE_JSON")
	if MODE == "PROD" {
		googleCredentials = defaultGoogleJson
	}
	client, err := storage.NewClient(ctx, option.WithCredentialsJSON([]byte(googleCredentials)))
	if err != nil {
		return "", err
	}
	defer client.Close()
	filename := strings.Split(gsUrl, "/")[3]
	opts := &storage.SignedURLOptions{
		Scheme:  storage.SigningSchemeV4,
		Method:  "GET",
		Expires: time.Now().Add(16 * time.Minute),
	}
	bucketName := os.Getenv("BUCKET_NAME")
	if MODE == "PROD" {
		bucketName = defaultBucketName
	}
	u, err := client.Bucket(bucketName).SignedURL(filename, opts)
	if err != nil {
		return "", err
	}
	return u, nil
}

func sendPayload(filePath string, fileInfo os.FileInfo) (string, string, error) {
	ctx := context.Background()
	googleCredentials := os.Getenv("GOOGLE_JSON")
	if MODE == "PROD" {
		googleCredentials = defaultGoogleJson
	}
	client, err := storage.NewClient(ctx, option.WithCredentialsJSON([]byte(googleCredentials)))
	if err != nil {
		return "", "", err
	}
	defer client.Close()
	bucketName := os.Getenv("BUCKET_NAME")
	if MODE == "PROD" {
		bucketName = defaultBucketName
	}
	bucket := client.Bucket(bucketName)
	var uploadPath string
	var uploadSize int64
	var isDir bool
	var tempTarPath string
	if fileInfo.IsDir() {
		isDir = true
		tempTarPath, uploadSize, err = createTarArchive(filePath)
		if err != nil {
			return "", "", err
		}
		uploadPath = tempTarPath
		defer os.Remove(tempTarPath)
	} else {
		uploadPath = filePath
		uploadSize = fileInfo.Size()
	}
	if uploadSize > 5*1024*1024*1024 {
		return "", "", fmt.Errorf("file size is too large: %d", uploadSize)
	}
	objName := fileInfo.Name()
	if isDir {
		objName = fileInfo.Name() + ".tar"
	}
	obj := bucket.Object(objName)
	w := obj.NewWriter(ctx)
	file, err := os.Open(uploadPath)
	if err != nil {
		return "", "", err
	}
	defer file.Close()
	bar := progressbar.DefaultBytes(uploadSize, "Sending")
	_, err = io.Copy(w, io.TeeReader(file, bar))
	if err != nil {
		return "", "", err
	}
	if err := w.Close(); err != nil {
		return "", "", err
	}

	gsUrl := fmt.Sprintf("gs://%s/%s", bucketName, objName)
	originalName := fileInfo.Name()
	return gsUrl, originalName, nil
}

func setKey(key string, url string, originalName string, isDir bool) error {
	authority := os.Getenv("AUTHORITY")
	if MODE == "PROD" {
		authority = defaultAuthority
	}
	authority = authority + "/drop"
	payload := fmt.Sprintf(`{"key":"%s","url":"%s","original_name":"%s","is_dir":%t}`, key, url, originalName, isDir)
	req, err := http.NewRequest("POST", authority, strings.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	secret := os.Getenv("SECRET")
	if MODE == "PROD" {
		secret = defaultSecret
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", secret))
	req.Header.Set("Content-Type", "application/json")
	resp, err := insecureClient.Do(req)
	if err != nil {
		return err
	}
	resBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	Response := sendResponse{}
	err = json.Unmarshal(resBody, &Response)
	if err != nil {
		return err
	}
	if Response.Status != "ok" {
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
	gsUrl, originalName, err := sendPayload(file, fileInfo)
	if err != nil {
		return err
	}
	key := getKey()
	signedUrl, err := generateSignedUrl(gsUrl, originalName)
	if err != nil {
		return err
	}
	if err = setKey(key, signedUrl, originalName, fileInfo.IsDir()); err != nil {
		return err
	}
	fmt.Println("Access key:", key)
	return nil
}
