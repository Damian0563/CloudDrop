package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/mdns"
	"github.com/urfave/cli/v3"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
)

func handleStream(conn net.Conn) error {
	reader := bufio.NewReader(conn)
	header, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	header = strings.TrimSpace(header)
	parts := strings.Split(header, ":")
	if len(parts) < 2 {
		return errors.New("invalid header format")
	}
	fileName := parts[0]
	fileSize, _ := strconv.ParseInt(parts[1], 10, 64)
	fmt.Printf("Receiving %s (%d bytes)...\n", fileName, fileSize)
	out, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.CopyN(out, reader, fileSize)
	return err
}

func Receive(ctx context.Context, c *cli.Command) error {
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
