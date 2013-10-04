package hoops

import (
        "encoding/json"
        "errors"
        "io"
        "io/ioutil"
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

func setFileField(h *ContributedHoop, name string, f multipart.File, fh *multipart.FileHeader, dataDir string) {
        fileName := h.getImageFilename(fh)
        path := filepath.Join(dataDir, fileName) 
        fCopy, err := os.Create(path)
        if err == nil {
                io.Copy(fCopy, f)
                h.Image = fileName
        }
}

func (h *ContributedHoop) FromRequest(r *http.Request, dataDir string) {
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
                        setFileField(h, "image", image, header, dataDir)
                }
        }
}

func (h *ContributedHoop) Save(dataDir string) error {
        filename := h.getFilenamePrefix() + ".json"
        path := filepath.Join(dataDir, filename) 
        jsonStr, err := json.Marshal(h)
        if err != nil {
                return err
        }
        return ioutil.WriteFile(path, jsonStr, 0600)
}
