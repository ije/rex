package main

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ije/rex"
)

const indexHTML = `
<h1>Welcome to use REX!</h1>
<p>Download the <a href="/nil.zip">nil.zip</a></p>
<p>Download the <a href="/main.js.zip">main.js.zip</a></p>
<p>Download the <a href="/www.zip">www.zip</a></p>
`

var errNotFound = errors.New("not found")

func main() {
	rex.Get("/", func(ctx *rex.Context) {
		ctx.HTML(indexHTML)
	})

	modtime := time.Now()

	rex.Get("/nil.zip", func(ctx *rex.Context) {
		zipContent, err := compress("../static/www/nil")
		if err != nil {
			if err == errNotFound {
				ctx.Error("file not found", 404)
			} else {
				ctx.Error(err.Error(), 500)
			}
			return
		}

		ctx.Content("application/zip", modtime, bytes.NewReader(zipContent))
	})

	rex.Get("/main.js.zip", func(ctx *rex.Context) {
		zipContent, err := compress("../static/www/main.js")
		if err != nil {
			if err == errNotFound {
				ctx.Error("file not found", 404)
			} else {
				ctx.Error(err.Error(), 500)
			}
			return
		}

		ctx.Content("application/zip", modtime, bytes.NewReader(zipContent))
	})

	rex.Get("/www.zip", func(ctx *rex.Context) {
		zipContent, err := compress("../static/www")
		if err != nil {
			if err == errNotFound {
				ctx.Error("file not found", 404)
			} else {
				ctx.Error(err.Error(), 500)
			}
			return
		}

		ctx.Content("application/zip", modtime, bytes.NewReader(zipContent))
	})

	rex.Use(rex.SendError())
	rex.Start(8080)
}

func compress(path string) (content []byte, err error) {
	fi, err := os.Stat(path)
	if err != nil {
		fmt.Println(err)
		if os.IsNotExist(err) {
			err = errNotFound
		}
		return
	}

	if fi.IsDir() {
		var dir string
		dir, err = filepath.Abs(path)
		if err != nil {
			return
		}

		buffer := bytes.NewBuffer(nil)
		archive := zip.NewWriter(buffer)

		err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			header, err := zip.FileInfoHeader(info)
			if err != nil {
				return err
			}

			header.Name = strings.TrimPrefix(strings.TrimPrefix(path, dir), "/")
			if header.Name == "" {
				return nil
			}

			if info.IsDir() {
				header.Name += "/"
			} else {
				header.Method = zip.Deflate
			}

			gzw, err := archive.CreateHeader(header)
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = io.Copy(gzw, file)
			return err
		})
		if err != nil {
			archive.Close()
			return
		}

		archive.Close()
		content = buffer.Bytes()
		return
	}

	header, err := zip.FileInfoHeader(fi)
	if err != nil {
		return
	}

	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	buffer := bytes.NewBuffer(nil)
	archive := zip.NewWriter(buffer)

	gzw, err := archive.CreateHeader(header)
	if err != nil {
		archive.Close()
		return
	}

	_, err = io.Copy(gzw, file)
	if err != nil {
		archive.Close()
		return
	}

	archive.Close()
	content = buffer.Bytes()
	return
}
