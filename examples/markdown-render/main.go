package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/ije/rex"
)

type Book struct {
	Title     string
	Author    string
	Published string
	Intro     string
	Slug      string
}

const listTplRaw = `
<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<title>Books</title>
</head>
<body>
	<h1>Books</h1>
	<ul>
		{{range .}}
			<li>
				<h2><a href="/p/{{.Slug}}">{{.Title}}</a></h2>
				<p>By {{.Author}}</p>
			</li>
		{{end}}
	</ul>
</body>
</html>
`

const pageTplRaw = `
<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<title>{{.Title}}</title>
</head>
<body>
	<h1>{{.Title}}</h1>
	<p>By {{.Author}}</p>
	<div>{{.Intro}}</div>
	<footer><a href="/">Back</a></footer>
</body>
`

var (
	books   = map[string]Book{}
	listTpl = rex.Tpl(listTplRaw)
	pageTpl = rex.Tpl(pageTplRaw)
)

func main() {
	rex.Use(rex.Compress())

	rex.GET("/{$}", func(ctx *rex.Context) any {
		return rex.Render(listTpl, books)
	})

	rex.GET("/p/{slug}", func(ctx *rex.Context) any {
		slug := ctx.PathValue("slug")
		if book, ok := books[slug]; ok {
			return rex.Render(pageTpl, book)
		}
		return rex.Status(404, "book not found")
	})

	<-rex.Start(context.Background(), 8080, func(port uint16) {
		fmt.Printf("Server running on http://localhost:%d\n", port)
	})
}

func init() {
	err := loadBooks("./books")
	if err != nil {
		panic("failed to load books: " + err.Error())
	}
	fmt.Println(len(books), "books loaded")
}

// load books (md files) from the given path
func loadBooks(path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(path + "/" + entry.Name())
		if err != nil {
			return err
		}
		book, err := parseBook(data)
		if err != nil {
			return err
		}
		book.Slug = strings.TrimSuffix(entry.Name(), ".md")
		books[book.Slug] = book
	}
	return nil
}

// parse book from markdown data
func parseBook(data []byte) (book Book, err error) {
	book = Book{}
	data = bytes.TrimSpace(data)
	// parse front matter if exists
	if bytes.HasPrefix(data, []byte("---")) {
		p := bytes.Split(data, []byte("---"))
		if len(p) > 2 {
			fm := p[1]
			data = bytes.Join(p[2:], []byte("---"))
			scanner := bufio.NewScanner(bytes.NewReader(fm))
			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "title:") {
					book.Title = strings.TrimSpace(line[6:])
				} else if strings.HasPrefix(line, "author:") {
					book.Author = strings.TrimSpace(line[7:])
				} else if strings.HasPrefix(line, "published:") {
					book.Published = strings.TrimSpace(line[11:])
				}
			}
			if err = scanner.Err(); err != nil {
				return
			}
		}

	}
	book.Intro, err = mdToHtml(data)
	return
}

func mdToHtml(data []byte) (string, error) {
	var html []string
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "## ") {
			html = append(html, "<h2>"+strings.TrimSpace(line[3:])+"</h2>")
		} else {
			html = append(html, "<p>"+resolveAnchorLinks(line)+"</p>")
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return strings.Join(html, "\n"), nil
}

var mdAnchorRegexp = regexp.MustCompile(`\[(.*?)\]\((.*?)\)`)

func resolveAnchorLinks(raw string) string {
	return mdAnchorRegexp.ReplaceAllString(raw, `<a href="$2">$1</a>`)
}
