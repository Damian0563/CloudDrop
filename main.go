package main

import (
	"context"
	"github.com/joho/godotenv"
	"github.com/urfave/cli/v3"
	"log"
	"os"
)

func main() {
	_ = godotenv.Load()
	file, err := os.OpenFile("timeout.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	app := &cli.Command{
		Name:    "clouddrop",
		Version: "0.2.0",
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
		},
	}
	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
