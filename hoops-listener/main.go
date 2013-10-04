package main

import (
        "encoding/json"
        "flag"
	"fmt"
        "mime"
	"net/http"

        "github.com/ghing/hoops"
)


var port int
var dataDir string

func hoopsHandler(w http.ResponseWriter, r *http.Request) {
        var hoopJSON []byte 
        v := r.Header.Get("Content-Type")
        d, _, err := mime.ParseMediaType(v)
        if err != nil || d != "multipart/form-data" {
                // TODO: Accept JSON
                http.Error(w, "Request must be encoded as multipart/form-data", http.StatusBadRequest)
                return
        }

        saver := hoops.FilesystemHoopSaver{DataDir:dataDir}
        mediaSaver := hoops.FilesystemHoopMediaSaver{DataDir:dataDir}
        hoop := hoops.NewContributedHoop()
        hoop.FromRequest(r)
        err = hoop.Save(hoops.HoopSaver(saver), hoops.HoopMediaSaver(mediaSaver))
        if err != nil {
                http.Error(w, "Error saving hoop", http.StatusInternalServerError)
                return
        }
        w.Header().Set("Content-Type", "application/json")
        hoopJSON, err = json.Marshal(hoop.Attributes())
        fmt.Fprintf(w, string(hoopJSON))
}

func init() {
        flag.IntVar(&port, "port", 8080, "port number")
        flag.StringVar(&dataDir, "data-dir", "", "Directory to store uploaded files")
}

func main() {
        flag.Parse()
	http.HandleFunc("/api/0.1/hoops/", hoopsHandler)
	http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}
