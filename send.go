package main

import (
	"archive/tar"
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
