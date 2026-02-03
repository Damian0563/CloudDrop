package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/mdns"
	"github.com/urfave/cli/v3"
	"io"
	"net"
	"os"
)

func Drop(ctx context.Context, filepath string, conn net.Conn) error {
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	header := fmt.Sprintf("%s:%d\n", info.Name(), info.Size())
	conn.Write([]byte(header))
	_, err = io.Copy(conn, file)
	fmt.Println("File dropped.")
	return err
}

func Send(ctx context.Context, c *cli.Command) error {
	if c.Args().Len() < 1 {
		return errors.New("you must provide a file path, no arguments provided")
	} else if c.Args().Len() > 1 {
		return errors.New("you can only provide one file path, too many arguments provided")
	}
	file := c.Args().First()
	if _, err := os.Stat(file); err != nil {
		return fmt.Errorf("file not found: %s", file)
	}
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
		fmt.Println("Dropping file and waiting for recevier...")
		return Drop(ctx, file, conn)
	}
}
