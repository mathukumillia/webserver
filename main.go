package main

import (
    "context"
    "log"
    "flag"
    "html/template"
    "io"
    "io/ioutil"
    "net/http"
    "os"
    "os/signal"
    "path"
    "syscall"
)

const (
    dirListingTmpl = "./public/templates/directory.html"
)

type FileLink struct {
    Name string
    Path string 
}

type DirPage struct {
    Title string
    Links []FileLink
}

var (
    fileDir string
)

func init(){
    flag.StringVar(&fileDir, "files", "", "The directory to search for files in")
}

func validateArgs() {
    if fileDir == "" {
        log.Fatalf("A file directory must be provided.")
    }
}

func handler(w http.ResponseWriter, r *http.Request) {
    filePath := fileDir + r.URL.Path
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
            for _, fileInfo := range fileInfos {
                link := FileLink{
                    Name: fileInfo.Name(),
                    Path: r.URL.Path + "/" + fileInfo.Name(),
                }
                links = append(links, link)
            }
            // Generate directory page and send it to the client.
            data := DirPage {
                Title: "Rudy's File Server",
                Links: links,
            }
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
                w.Header().Set("Content-Disposition", "attachment; filename=" + path.Base(filePath))
                io.Copy(w, file)
            }
        }
    }
}

func main() {
    flag.Parse()
    validateArgs()
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    srvMux := http.NewServeMux()
    // Add handler for the home page
    srvMux.HandleFunc("/", handler)
    srv := http.Server{
        Addr: ":8080",
        Handler: srvMux,
    }

    // Enforce clean shutdown on SIGTERM and SIGINT
    signals := make(chan os.Signal)
    signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
    go func() {
        select {
            case <-signals:
                log.Printf("Shutting down file server...")
                srv.Shutdown(ctx)
                cancel()
            case <-ctx.Done():
        }
    }()

    err := srv.ListenAndServe()
    if err != nil {
        log.Printf("%v", err)
    }
}
