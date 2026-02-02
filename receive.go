package main

import (
	"context"
	"fmt"
	"github.com/urfave/cli/v3"
)

func Receive(ctx context.Context, c *cli.Command) error {
	fmt.Println("Receiving files...")
	return nil
}
