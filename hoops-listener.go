package main

import (
        "encoding/json"
        "errors"
        "flag"
	"fmt"
        "io"
        "io/ioutil"
        "mime"
        "mime/multipart"
	"net/http"
        "os"
        "path/filepath"
        "reflect"
        "strconv"
        "strings"
        "time"
        "github.com/nu7hatch/gouuid"
)

type ContributedHoop struct {
        Id string
        Location string
        Lat float64
        Lng float64
        Image string
        Story []byte
        ContactOk bool
        Email string
        Phone string
        Created time.Time
}

func NewContributedHoop() *ContributedHoop {
        u4, err := uuid.NewV4()
        h := ContributedHoop{}
        if err == nil {
                h.Id = u4.String()
        }
        h.Created = time.Now()
        return &h
}

func setField(h *ContributedHoop, name string, value string) {
        s := reflect.ValueOf(h).Elem()
        f := s.FieldByName(name)
        switch f.Type().Kind() {
        case reflect.Bool:
                parsedVal, err := strconv.ParseBool(value)
                if err == nil {
                        f.SetBool(parsedVal)
                }
        case reflect.Float32, reflect.Float64:
                parsedVal, err := strconv.ParseFloat(value, 32)
                if err == nil {
                        f.SetFloat(parsedVal)
                }
        case reflect.Slice:
                f.SetBytes([]byte(value))
        default:
                f.SetString(value)
        }
}

func typeToExt(contentType string) (ext string, err error) {
        err = nil
        ext = ""
        switch contentType {
        case "image/png":
                ext = ".png"
        case "image/jpeg":
                ext = ".jpg"
        default:
                err =  errors.New("Unknown MIME type")
        }
        return
}

func (h *ContributedHoop) getFilenamePrefix() (string) {
        const layout = "2006-01-02"
        dateStr := strings.Replace(h.Created.Format(layout), "-", "", -1)
        id := strings.Replace(h.Id, "-", "", -1)
        return dateStr + "-" + id
}

func (h *ContributedHoop) getImageFilename(fh *multipart.FileHeader) (string) {
        ext, err := typeToExt(fh.Header.Get("Content-Type"))
        if err != nil {
                // TODO: Handle error
        }
        return h.getFilenamePrefix() + ext
}

func setFileField(h *ContributedHoop, name string, f multipart.File, fh *multipart.FileHeader) {
        fileName := h.getImageFilename(fh)
        path := filepath.Join(dataDir, fileName) 
        fCopy, err := os.Create(path)
        if err == nil {
                io.Copy(fCopy, f)
                h.Image = fileName
        }
}

func (h *ContributedHoop) fromRequest(r *http.Request) {
        // TODO: Use reflect to iterate over the fields more dynamically
        // See http://blog.golang.org/laws-of-reflection
        setField(h, "Location", r.FormValue("location"))
        setField(h, "Lat", r.FormValue("lat"))
        setField(h, "Lng", r.FormValue("lng"))
        setField(h, "Story", r.FormValue("story"))
        setField(h, "ContactOk", r.FormValue("contact-ok"))
        setField(h, "Email", r.FormValue("email"))
        setField(h, "Phone", r.FormValue("phone"))
        image, header, err := r.FormFile("image")
        if err == nil {
                mtype := header.Header.Get("Content-Type")
                if mtype == "image/jpeg" || mtype == "image/png" {
                        setFileField(h, "image", image, header)
                }
        }
}

func (h *ContributedHoop) save() error {
        filename := h.getFilenamePrefix() + ".json"
        path := filepath.Join(dataDir, filename) 
        jsonStr, err := json.Marshal(h)
        if err != nil {
                return err
        }
        return ioutil.WriteFile(path, jsonStr, 0600)
}

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

        hoop := NewContributedHoop()
        hoop.fromRequest(r)
        err = hoop.save()
        if err != nil {
                http.Error(w, "Error saving hoop", http.StatusInternalServerError)
                return
        }
        w.Header().Set("Content-Type", "application/json")
        hoopJSON, err = json.Marshal(hoop)
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
