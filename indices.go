package main

import (
	"bytes"
	"gopkg.in/russross/blackfriday.v2"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"regexp"
	// uses text/template to allow for scripts or HTML in .md files
	"text/template"
)

type Index struct {
	Path        string
	Title       string
	Content     string
	VisiblePath string
	NavItems    map[string]string
}

type Item struct {
	Path        string
	Title       string
	Content     string
	VisiblePath string
}

var indices map[string]*Index
var items []Item

// reads a mardkown file, gets HTML from blackfriday, and returns the HTML and
// title
func markdownToHtml(path string, info os.FileInfo) (string, string) {
	markdown, err := ioutil.ReadFile(path)
	markdown = bytes.Replace(markdown, []byte("\r"), nil, -1)
	lines := strings.Split(string(markdown), "\n")

	unsafeTitle := lines[0]
	regex, err := regexp.Compile("[^a-zA-Z0-9 .'-]+")
	if err != nil {
		panic(err)
	}
	title := regex.ReplaceAllString(unsafeTitle, "")
	if []rune(title)[0] == ' ' {
		title = strings.Replace(title, " ", "", 1)
	}
	if err != nil {
		panic(err)
	}
	var html []byte = blackfriday.Run(markdown, blackfriday.WithExtensions(blackfriday.BackslashLineBreak))
	return string(html), title

}

func replaceSuffix(str string, old string, new string) string {
	str = strings.TrimRight(str, old)
	return str + new
}

func copyFile(srcPath string, dstPath string) int64 {
	reader, err := os.Open(srcPath)
	if err != nil {
		panic(err)
	}
	defer reader.Close()

	writer, err := os.Create(dstPath)
	if err != nil {
		panic(err)
	}
	defer writer.Close()

	length, err := io.Copy(writer, reader)
	if err != nil {
		panic(err)
	}
	return length
}

func processFilesIn(dir string) {
	indices = make(map[string]*Index)
	items = []Item{}
	processFile := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			panic(err)
		}
		path = filepath.ToSlash(path)
		staticPath := strings.Replace(path, dir, dir+"_static", 1)
		relPath := strings.Split(path, dir)[1]
		visiblePath := strings.TrimRight(relPath, info.Name())
		// handle root directory visibility
		if visiblePath == "" {
			visiblePath = "/"
		} else if info.IsDir() {
			// if this isn't the root but is a directory, the visiblePath should include
			// its name
			visiblePath = visiblePath + info.Name() + "/"
		}
		// create an "empty" index for every directory
		if info.IsDir() {
			err := os.Mkdir(staticPath, os.ModePerm)
			if err != nil {
				panic(err)
			}
			index := Index{
				relPath + "/index.html",
				"Filed in " + visiblePath,
				"",
				visiblePath,
				make(map[string]string, 0)}
			indices[visiblePath] = &index
			if visiblePath != "/" {
				parentRelPath := filepath.Dir(filepath.Dir(visiblePath))
				parentVisiblePath := filepath.ToSlash(parentRelPath)
				indices[parentVisiblePath].NavItems[relPath + "/index.html"] = "Filed in " + visiblePath
			}
		} else if filepath.Ext(info.Name()) == ".md" {
			// this is a markdown file; send it off
			var htmlPath string
			html, title := markdownToHtml(path, info)
			if info.Name() == "index.md" {
				htmlPath = replaceSuffix(relPath, ".md", ".html")
				indices[visiblePath].Path = htmlPath
				indices[visiblePath].Title = title
				indices[visiblePath].Content = html
				indices[visiblePath].VisiblePath = visiblePath
				if visiblePath != "/" {
					parentRelPath := filepath.Dir(filepath.Dir(visiblePath))
			  	parentVisiblePath := filepath.ToSlash(parentRelPath)
					indices[parentVisiblePath].NavItems[htmlPath] = title
				}
			} else {
				htmlPath = replaceSuffix(relPath, ".md", "/index.html")
				// make a new folder and index.html for clean URLs
				err := os.Mkdir(strings.TrimRight(staticPath, ".md"), os.ModePerm)
				if err != nil {
					panic(err)
				}
				item := Item{
					htmlPath,
					title,
					html,
					visiblePath}
				items = append(items, item)
			  indices[visiblePath].NavItems[htmlPath] = item.Title
			}
			if err != nil {
				panic(err)
			}
		} else {
			// if it's not markdown or a directory, include it in the final static site
			// note: it might be overwritten by another file!
			copyFile(path, staticPath)
		}
		return nil
	}
	// prepare structs
	err := filepath.Walk(dir, processFile)
	if err != nil {
		panic(err)
	}
}

func writeHtml(dir string) {
	var indexHtml = `
	<!DOCTYPE html>
	<html lang=en>
	<head>
	  <title>{{.Title}}</title>
	  <meta name="viewport" content="width=device-width">
	  <link rel=icon href=data:,>
	  <style>
	  body {
	    font-family: Verdana, Arial, sans-serif;
	    color: #3b3837;
	  }
	  main {
	    max-width: 70ch;
	    padding: 2ch;
	    margin: auto;
	  }
		nav {
			list-style-type: none;
			padding: 1rem 0;
		}
	  a {
	    text-decoration: none;
	    outline: 0;
	  }
	  a:hover {
	    text-decoration: underline;
	  }
	  ::selection {
	    background-color: #e5e5e5;
	  }
	  </style>
	</head>
		<body>
			<main>
				{{.Content}}
				</hr>
				<nav>
					<h2>Filed in <a href="{{.VisiblePath}}">{{.VisiblePath}}</a></h2>
					{{ range $Path, $Title := .NavItems }}
						<li><a href="{{ $Path }}">{{ $Title }}</a></li>
					{{end}}
				</nav>
			</main>
		</body>
	</html>
	`

	var itemHtml = `
	<!DOCTYPE html>
	<html lang=en>
	<head>
		<title>{{.Title}}</title>
		<meta name="viewport" content="width=device-width">
		<link rel=icon href=data:,>
		<style>
		body {
			font-family: Verdana, Arial, sans-serif;
			color: #3b3837;
		}
		main {
			max-width: 70ch;
			padding: 2ch;
			margin: auto;
		}
		a {
			text-decoration: none;
			outline: 0;
		}
		a:hover {
			text-decoration: underline;
		}
		::selection {
			background-color: #e5e5e5;
		}
		</style>
	</head>
		<body>
			<main>
				{{.Content}}
				<h4>Filed in <a href="{{.VisiblePath}}">{{.VisiblePath}}</a></h4>
			</main>
		</body>
	</html>
	`

	for _, item := range items {
		tmpl, err := template.New("item").Parse(itemHtml)
		if err != nil {
			panic(err)
		}

		writer, err := os.Create(dir+item.Path)
		if err != nil {
			panic(err)
		}

		err = tmpl.Execute(writer, item)
		if err != nil {
			panic(err)
		}

	}

	for _, index := range indices {
		tmpl, err := template.New("index").Parse(indexHtml)
		if err != nil {
			panic(err)
		}

		writer, err := os.Create(dir+index.Path)
		if err != nil {
			panic(err)
		}

		err = tmpl.Execute(writer, index)
		if err != nil {
			panic(err)
		}

	}

}

func main() {
	args := os.Args
	if len(args) != 2 {
		panic("Please provide exactly one argument, the directory with your content.")
	}
	dir := filepath.ToSlash(args[1])
	processFilesIn(dir)
	staticDir := dir + "_static"
	writeHtml(staticDir)
}