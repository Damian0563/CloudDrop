package main

import (
	"archive/tar"
	"context"
	"fmt"
	"github.com/hashicorp/mdns"
	"github.com/schollz/progressbar/v3"
	"github.com/urfave/cli/v3"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
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
