package main

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"github.com/joho/godotenv"
	"github.com/urfave/cli/v3"
	"log"
	"net/http"
	"os"
)

var insecureClient = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
}
var defaultGoogleJson string
var defaultSecret string
var defaultBucketName string
var defaultAuthority string
var MODE string = "PROD"

func main() {
	_ = godotenv.Load()
	if decoded, err := base64.StdEncoding.DecodeString(defaultGoogleJson); err == nil {
		var prettyJSON map[string]interface{}
		if json.Unmarshal(decoded, &prettyJSON) == nil {
			if prettyBytes, err := json.Marshal(prettyJSON); err == nil {
				defaultGoogleJson = string(prettyBytes)
			}
		}
	}
	file, err := os.OpenFile("timeout.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	app := &cli.Command{
		Name:    "clouddrop",
		Version: "v1.0",
		Usage:   "CloudDrop is a simple CLI tool to upload files to cloud storage services.",
		Commands: []*cli.Command{
			{
				Name:   "drop",
				Usage:  "Upload files to P2P network. Must provide a valid file path to a file or directory.",
				Action: Send,
			},
			{
				Name:   "pick",
				Usage:  "Receive files from P2P network. No arguments required.",
				Action: Receive,
			},
			{
				Name:   "send",
				Usage:  "Upload files over public internet. Must provide a valid file path.",
				Action: superSend,
			},
			{
				Name:   "receive",
				Usage:  "Receive files over public internet. Must provide a valid code received from the sender.",
				Action: superReceive,
			},
			{
				Name:  "info",
				Usage: "Display useful information about limits of use.",
				Action: func(ctx context.Context, c *cli.Command) error {
					log.Println("CloudDrop is a simple CLI tool to upload files to cloud storage services.")
					log.Println("Please keep up to date with latest releases at https://github.com/Damian0563/CloudDrop/releases")
					log.Println("")
					log.Println("--- Usage ---")
					log.Println("CloudDrop allows you to share files in two ways:")
					log.Println("1. Cloud-Based (Public Internet): Skip the middleman with this free CLI tool that doesn't store your data.")
					log.Println("2. P2P (Local Network): Share files directly with friends or family—both clients need CloudDrop CLI installed.")
					log.Println("")
					log.Println("--- Commands ---")
					log.Println("  drop     - Upload files to P2P network. Must provide a valid file path to a file or directory.")
					log.Println("  pick     - Receive files from P2P network. No arguments required.")
					log.Println("  send     - Upload files over public internet. Must provide a valid file path.")
					log.Println("  receive  - Receive files over public internet. Must provide a valid code received from the sender.")
					log.Println("")
					log.Println("--- Limits ---")
					log.Println("  File expiry (Public Internet): 5 minutes after code generation")
					log.Println("  File expiry (P2P): No expiry (real-time transfer)")
					log.Println("  Maximum instance size (send method): 5GB")
					return nil
				},
			},
		},
	}
	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
