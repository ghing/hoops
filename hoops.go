package hoops

import (
	"encoding/json"
	"errors"
	"github.com/nu7hatch/gouuid"
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
)

type HoopsConfig struct {
	Port                 int
	DataDir              string
	EmailSendingEmail    string
	EmailSendingUsername string
	EmailSendingPassword string
	EmailSendingHost     string
	NotificationEmail    string
	OAuthClientId        string
	OAuthClientSecret    string
	OAuthTokenCacheFile  string
	SpreadsheetKey       string
}

func ParseConfig(filename string, c *HoopsConfig) error {
	jsonData, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	err = json.Unmarshal(jsonData, c)
	if err != nil {
		return err
	}
	return nil
}

type HoopAttributes struct {
	Id        string
	Location  string
	Lat       float64
	Lng       float64
	Image     string
	Story     string
	ContactOk bool
	Email     string
	Phone     string
	Created   time.Time
}

type Hoop interface {
	Attributes() HoopAttributes
	Id() string
	Created() time.Time
	Read(reader HoopReader) error
	Save(saver HoopSaver) error
	SaveMedia(mediaSaver HoopMediaSaver) error
	MarshalJSON() ([]byte, error)
	UnmarshalJSON([]byte) error
}

type HoopReader interface {
	Read(h *Hoop) error
}

type HoopSaver interface {
	Save(h Hoop) error
}

type HoopMediaSaver interface {
	Save(h Hoop, f multipart.File, fh *multipart.FileHeader) (string, error)
}

func getFilenamePrefix(h Hoop) string {
	const layout = "2006-01-02"
	dateStr := strings.Replace(h.Created().Format(layout), "-", "", -1)
	id := strings.Replace(h.Id(), "-", "", -1)
	return dateStr + "-" + id
}

type FilesystemHoopReader struct {
	DataDir string
}

func (r FilesystemHoopReader) Read(h *Hoop) error {
	filename := getFilenamePrefix(*h) + ".json"
	path := filepath.Join(r.DataDir, filename)
	return r.ReadFromFile(h, path)
}

func (r FilesystemHoopReader) ReadFromFile(h *Hoop, filename string) error {
	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	err = json.Unmarshal(contents, *h)
	if err != nil {
		return err
	}
	return nil
}

type FilesystemHoopSaver struct {
	DataDir string
}

func (s FilesystemHoopSaver) Save(h Hoop) error {
	filename := getFilenamePrefix(h) + ".json"
	path := filepath.Join(s.DataDir, filename)
	jsonStr, err := json.Marshal(h)
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
		err = errors.New("Unknown MIME type")
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
	attributes      HoopAttributes
	imageFile       multipart.File
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

func (h *ContributedHoop) Read(reader HoopReader) error {
	hoop := Hoop(h)
	return reader.Read(&hoop)
}

func (h *ContributedHoop) Save(saver HoopSaver) error {
	return saver.Save(Hoop(h))
}

func (h *ContributedHoop) SaveMedia(mediaSaver HoopMediaSaver) error {
	if h.imageFile != nil {
		filename, err := mediaSaver.Save(Hoop(h), h.imageFile, h.imageFileHeader)
		if err == nil {
			h.attributes.Image = filename
		}
	}

	return nil
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

func (h *ContributedHoop) SetField(name string, value string) {
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
	h.SetField("Location", r.FormValue("location"))
	h.SetField("Lat", r.FormValue("lat"))
	h.SetField("Lng", r.FormValue("lng"))
	h.SetField("Story", r.FormValue("story"))
	h.SetField("ContactOk", r.FormValue("contact-ok"))
	h.SetField("Email", r.FormValue("email"))
	h.SetField("Phone", r.FormValue("phone"))
	image, header, err := r.FormFile("image")
	if err == nil {
		mtype := header.Header.Get("Content-Type")
		if mtype == "image/jpeg" || mtype == "image/png" {
			h.imageFile = image
			h.imageFileHeader = header
		}
	}
}

func (h *ContributedHoop) MarshalJSON() ([]byte, error) {
	return json.Marshal(h.Attributes())
}

func (h *ContributedHoop) UnmarshalJSON(j []byte) error {
	return json.Unmarshal(j, &h.attributes)
}
