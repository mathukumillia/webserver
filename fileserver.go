package main

import (
	"context"
	"database/sql"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
)

type FileLink struct {
	Name string
	Path string
}

type DirPage struct {
	Title string
	Links []FileLink
}

func (f *Fileserver) reqHandler(w http.ResponseWriter, r *http.Request) {
	filePath := f.fileDir + r.URL.Path
	pathInfo, err := os.Stat(filePath)
	if err != nil {
		log.Printf("error accessing path: %s, %v", filePath, err)
		w.WriteHeader(http.StatusNotFound)
	} else {
		switch mode := pathInfo.Mode(); {
		case mode.IsDir():
			// Enumerate the files in the directory.
			fileInfos, err := ioutil.ReadDir(filePath)
			if err != nil {
				log.Printf("error reading directory: %s, %v", filePath, err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			var links []FileLink
			var pathWithSlash string = r.URL.Path
			if pathWithSlash[len(pathWithSlash)-1:] != "/" {
				pathWithSlash += "/"
			}
			for _, fileInfo := range fileInfos {
				link := FileLink{
					Name: fileInfo.Name(),
					Path: pathWithSlash + fileInfo.Name(),
				}
				links = append(links, link)
			}
			// Generate directory page and send it to the client.
			data := DirPage{
				Title: "Rudy's File Server",
				Links: links,
			}
			dirListingTmpl := f.templateDir + "/directory.html"
			tmpl, err := template.ParseFiles(dirListingTmpl)
			if err != nil {
				log.Printf("Failed loading directory template: %v", err)
				return
			}
			w.WriteHeader(http.StatusOK)
			err = tmpl.Execute(w, data)
			if err != nil {
				log.Printf("Failed executing directory template: %v", err)
			}
		case mode.IsRegular():
			// Send file to the client.
			file, err := os.Open(filePath)
			if err != nil {
				log.Printf("error opening file: %s, %v", filePath, err)
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				w.Header().Set("Content-Disposition", "attachment; filename="+path.Base(filePath))
				io.Copy(w, file)
			}
		}
	}
}

type Fileserver struct {
	fileDir     string
	srv         http.Server
	dbConn      *sql.DB
	templateDir string
}

func NewFileServer(addr, fileDir, templateDir string) Fileserver {
	srvMux := http.NewServeMux()
	fileSrv := Fileserver{
		fileDir: fileDir,
		srv: http.Server{
			Addr:    addr,
			Handler: srvMux,
		},
		dbConn:      nil,
		templateDir: templateDir,
	}
	srvMux.HandleFunc("/", fileSrv.reqHandler)
	return fileSrv
}

func (f *Fileserver) Start() error {
	return f.srv.ListenAndServe()
}

func (f *Fileserver) Stop(ctx context.Context) error {
	return f.srv.Shutdown(ctx)
}
