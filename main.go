package main

import (
    "context"
    "encoding/json"
    "log"
    "flag"
    "io"
    "io/ioutil"
    "net/http"
    "os"
    "os/signal"
    "path"
    "syscall"
)

const (
    indexPath = "./public/index.html"
)

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
    if r.URL.Path == "/" {
        file, err := os.Open(indexPath)        
        if err != nil {
            log.Printf("error loading index.html: %v", err)
            w.WriteHeader(http.StatusInternalServerError)
        } else {
            w.WriteHeader(http.StatusOK)
        }
        io.Copy(w, file)
    } else {
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
                } else {
                    w.WriteHeader(http.StatusOK)
                }
                var names []string
                for _, fileInfo := range fileInfos {
                    names = append(names, fileInfo.Name())
                }
                // Encode file names in json and send them to the client.
                enc := json.NewEncoder(w)
                enc.Encode(names)
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
