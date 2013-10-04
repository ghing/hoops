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

type HoopAttributes struct {
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

type Hoop interface {
        Attributes() HoopAttributes
        Id() string
        Created() time.Time
        Save(saver HoopSaver, mediaSaver HoopMediaSaver) error
}

type HoopSaver interface {
        Save(h Hoop) error
}

type HoopMediaSaver interface {
        Save(h Hoop, f multipart.File, fh *multipart.FileHeader) (string, error)
}

func getFilenamePrefix(h Hoop) (string) {
        const layout = "2006-01-02"
        dateStr := strings.Replace(h.Created().Format(layout), "-", "", -1)
        id := strings.Replace(h.Id(), "-", "", -1)
        return dateStr + "-" + id
}


type FilesystemHoopSaver struct {
        DataDir string
}

func (s FilesystemHoopSaver) Save(h Hoop) error {
        filename := getFilenamePrefix(h) + ".json"
        path := filepath.Join(s.DataDir, filename) 
        jsonStr, err := json.Marshal(h.Attributes())
        if err != nil {
                return err
        }
        return ioutil.WriteFile(path, jsonStr, 0600)
}

type FilesystemHoopMediaSaver struct {
        DataDir string
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

func imageFilename(h Hoop, fh *multipart.FileHeader) string {
        ext, err := typeToExt(fh.Header.Get("Content-Type"))
        if err != nil {
                // TODO: Handle error
        }
        return getFilenamePrefix(h) + ext
}

func (s FilesystemHoopMediaSaver) Save(h Hoop, f multipart.File, fh *multipart.FileHeader) (filename string, err error) {
        err = nil
        filename = imageFilename(h, fh)
        path := filepath.Join(s.DataDir, filename)
        fCopy, err := os.Create(path)
        if err != nil {
                return
        }

        io.Copy(fCopy, f)
        return
}


type ContributedHoop struct {
        attributes HoopAttributes
        imageFile multipart.File
        imageFileHeader *multipart.FileHeader
}

func NewContributedHoop() *ContributedHoop {
        u4, err := uuid.NewV4()
        h := ContributedHoop{}
        h.attributes = HoopAttributes{}
        if err == nil {
                h.attributes.Id = u4.String()
        }
        h.attributes.Created = time.Now()
        return &h
}

func (h *ContributedHoop) Save(saver HoopSaver, mediaSaver HoopMediaSaver) error {
        filename, err := mediaSaver.Save(Hoop(h), h.imageFile, h.imageFileHeader)
        if err == nil {
                h.attributes.Image = filename
        }

        return saver.Save(Hoop(h))
}

func (h *ContributedHoop) Id() string {
        return h.attributes.Id
}

func (h *ContributedHoop) Created() time.Time {
        return h.attributes.Created
}

func (h *ContributedHoop) Image() string {
        return h.attributes.Image
}

func (h *ContributedHoop) Attributes() HoopAttributes {
        return h.attributes
}

func (h *ContributedHoop) setField(name string, value string) {
        s := reflect.ValueOf(&h.attributes).Elem()
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

func (h *ContributedHoop) FromRequest(r *http.Request) {
        // TODO: Use reflect to iterate over the fields more dynamically
        // See http://blog.golang.org/laws-of-reflection
        h.setField("Location", r.FormValue("location"))
        h.setField("Lat", r.FormValue("lat"))
        h.setField("Lng", r.FormValue("lng"))
        h.setField("Story", r.FormValue("story"))
        h.setField("ContactOk", r.FormValue("contact-ok"))
        h.setField("Email", r.FormValue("email"))
        h.setField("Phone", r.FormValue("phone"))
        image, header, err := r.FormFile("image")
        if err == nil {
                mtype := header.Header.Get("Content-Type")
                if mtype == "image/jpeg" || mtype == "image/png" {
                        h.imageFile = image
                        h.imageFileHeader = header
                }
        }
}
