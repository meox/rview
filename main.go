package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gabriel-vasile/mimetype"
)

type entry_info struct {
	name string
	path string
	date time.Time
}

func main() {
	root := flag.String("r", ".", "root path")
	port := flag.Uint("p", 5555, "service port")
	listenAddr := flag.String("l", "", "listen address")
	mode := flag.String("m", "lexy", "sorting order: lexy, lastmod, bydate")
	filter := flag.String("f", "video", "filter")
	flag.Parse()

	if root == nil {
		log.Fatal("invalid root")
	}
	if port == nil {
		log.Fatal("invalid port")
	}
	if listenAddr == nil {
		log.Fatal("invalid listen address")
	}
	if mode == nil {
		log.Fatal("invalid mode")
	}
	if filter == nil {
		log.Fatal("invalid filter")
	}

	entries := retrieveFiles(*root, *mode, *filter)
	data := populatePage(entries)

	// setup server
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fpath, _ := filepath.Abs(*root)
		fmt.Fprint(w, "<!DOCTYPE html>\n<html>\n")
		fmt.Fprint(w, "<head>\n")
		fmt.Fprint(w, "<style type=\"text/css\">\n")
		fmt.Fprint(w, `
a {
	color: hotpink;
}
table {
	min-width: 50%;
}
th {
	color: #000;
	background-color: #eee;
}
td {
	color: #ccc;
	background-color: #111;
}
`)
		fmt.Fprintln(w, "</style>")
		fmt.Fprintln(w, "</head>")

		fmt.Fprint(w, "<body style=\"color:white;background:black\">\n")
		fmt.Fprintf(w, "<h3 style=\"color:yellow\">%s</h3>\n", fpath + filterTitle(*filter))

		fmt.Fprintln(w, "<table>")
		fmt.Fprintln(w, "<tr><th>Name</th><th>Date</th></tr>")
		fmt.Fprint(w, data)
		fmt.Fprintln(w, "</table>")

		fmt.Fprintln(w, "</body>")
		fmt.Fprintf(w, "</html>")
	})

	http.HandleFunc("/content/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Query().Get("path")
		log.Printf("request: %s", path)
		data, err := os.ReadFile(path)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Write(data)
	})

	log.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%d", *listenAddr, *port), nil))
}

func retrieveFiles(root string, mode string, filter string) []entry_info {
	var rs []entry_info

	// walk dir
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		mtype, err := mimetype.DetectFile(path)
		if err != nil {
			fmt.Printf("WARN: %s\n", err)
			return nil
		}
		fileType := mtype.String()

		if filter == "" || (filter != "" && strings.Contains(fileType, filter)) {
			info, err := d.Info()
			if err != nil {
				return nil
			}
			rs = append(rs, entry_info{
				name: d.Name(),
				path: path,
				date: info.ModTime(),
			})
		}
		return nil
	})

	// sort it!
	switch mode {
	case "lexy":
		return rs
	case "lastmod":
		sort.Slice(rs, func(i, j int) bool {
			return rs[i].date.After(rs[j].date)
		})
	case "bydate":
		sort.Slice(rs, func(i, j int) bool {
			return rs[i].date.Before(rs[j].date)
		})
	}

	return rs
}

func populatePage(entries []entry_info) string {
	var w strings.Builder
	for _, e := range entries {
		w.WriteString(entry(e.path, e.name, e.date.Format("2006-01-02 15:04:05")))
		w.WriteString("\n")
	}

	return w.String()
}

func entry(filePath string, name string, date string) string {
	params := url.Values{}
	params.Add("path", filePath)
	e := fmt.Sprintf("<tr><td><a href=\"/content/?%s\">%s</a></td>", params.Encode(), name)
	e += fmt.Sprintf("<td>%s</td></tr>", date)
	return e
}

func filterTitle(f string) string {
	if f == "" {
		return ""
	}

	return fmt.Sprintf(", with filter: %s", f)
}