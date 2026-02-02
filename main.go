package main

import (
	"context"
	"github.com/urfave/cli/v3"
	"log"
	"os"
)

func main() {
	app := &cli.Command{
		Name:  "clouddrop",
		Usage: "CloudDrop is a simple CLI tool to upload files to cloud storage services.",
		Commands: []*cli.Command{
			{
				Name:   "drop",
				Usage:  "Upload files to cloud storage services.",
				Action: Send,
			},
			{
				Name:   "receive",
				Usage:  "Receive files from cloud storage services.",
				Action: Receive,
			},
		},
	}
	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
