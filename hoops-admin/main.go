package main

import (
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"code.google.com/p/goauth2/oauth"

	"github.com/ghing/hoops"
)

var conf hoops.HoopsConfig
var configFilename string
var oauthCode string

func init() {
	conf = hoops.HoopsConfig{}
	flag.StringVar(&configFilename, "config", "", "Configuration file")
	flag.StringVar(&oauthCode, "code", "", "OAuth 2 Authorization Code")
}

func getOAuthConfig() *oauth.Config {
	return &oauth.Config{
		ClientId:     conf.OAuthClientId,
		ClientSecret: conf.OAuthClientSecret,
		RedirectURL:  "http://apps.terrorware.com/hoops/",
		Scope:        "https://spreadsheets.google.com/feeds https://docs.google.com/feeds",
		AuthURL:      "https://accounts.google.com/o/oauth2/auth",
		TokenURL:     "https://accounts.google.com/o/oauth2/token",
		TokenCache:   oauth.CacheFile(conf.OAuthTokenCacheFile),
		AccessType:   "offline",
	}
}

func getOAuthToken() {
	// Set up a configuration.
	oauthConfig := getOAuthConfig()

	if oauthCode == "" {
		url := oauthConfig.AuthCodeURL("")
		fmt.Println("Visit this URL to get a code, then run again with -code=YOUR_CODE\n")
		fmt.Println(url)
		return
	}

	// Set up a Transport using the config.
	transport := &oauth.Transport{Config: oauthConfig}

	_, err := transport.Exchange(oauthCode)
	if err != nil {
		log.Fatal("Exchange:", err)
	}

	// (The Exchange method will automatically cache the token.)
	fmt.Printf("Token is cached in %v\n", oauthConfig.TokenCache)
}

func getClient() (*http.Client, error) {
	oauthConfig := getOAuthConfig()
	transport := &oauth.Transport{Config: oauthConfig}

	token, err := oauthConfig.TokenCache.Token()
	if err != nil {
		return nil, errors.New("Unable to get OAuth token. Run the oauthtoken command first")
	}

	transport.Token = token

	return transport.Client(), nil
}

func get(c *http.Client, url string) ([]byte, error) {
	r, err := c.Get(url)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	return ioutil.ReadAll(r.Body)
}

func post(c *http.Client, url string, bodyType string, body string) ([]byte, error) {
	r, err := c.Post(url, bodyType, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	return ioutil.ReadAll(r.Body)
}

type Link struct {
	Rel  string `xml:"rel,attr"`
	Href string `xml:"href,attr"`
}

type Entry struct {
	Link []Link `xml:"link"`
}

type Feed struct {
	XMLName xml.Name `xml:"feed"`
	Link    []Link   `xml:"link"`
	Entry   []Entry  `xml:"entry"`
}

func getFeed(client *http.Client, url string, feed *Feed) error {
	body, err := get(client, url)

	if err != nil {
		return err
	}

	return xml.Unmarshal(body, &feed)
}

func getUrls(links []Link) map[string]string {
	urls := make(map[string]string)
	for _, link := range links {
		urls[link.Rel] = link.Href
	}
	return urls
}

func getURL(urls map[string]string, key string) (string, error) {
	for rel, url := range urls {
		if rel == key {
			return url, nil
		}
	}

	return "", errors.New("Unable to retrieve URL")
}

func getAddRowURL(urls map[string]string) (string, error) {
	return getURL(urls, "http://schemas.google.com/g/2005#post")
}

func getListFeedURL(urls map[string]string) (string, error) {
	return getURL(urls, "http://schemas.google.com/spreadsheets/2006#listfeed")
}

func getWorksheetUrls(client *http.Client, worksheetIndex int) (map[string]string, error) {
	feed := Feed{}
	requestURL := fmt.Sprintf("https://spreadsheets.google.com/feeds/worksheets/%s/private/full", conf.SpreadsheetKey)
	err := getFeed(client, requestURL, &feed)

	if err != nil {
		return nil, errors.New("Request failed")
	}

	entry := feed.Entry[worksheetIndex]
	urls := getUrls(entry.Link)

	listFeedURL, err := getListFeedURL(urls)
	err = getFeed(client, listFeedURL, &feed)
	if err != nil {
		return nil, err
	}

	return getUrls(feed.Link), nil
}

func rowXml(h hoops.Hoop) string {
	attrs := h.Attributes()
	template := `
               <entry xmlns="http://www.w3.org/2005/Atom"
                      xmlns:gsx="http://schemas.google.com/spreadsheets/2006/extended">
                   <gsx:id>%s</gsx:id>
                   <gsx:location>%s</gsx:location>
                   <gsx:lat>%f</gsx:lat>
                   <gsx:lng>%f</gsx:lng>
                   <gsx:image>%s</gsx:image>
                   <gsx:story>%s</gsx:story>
                   <gsx:contactok>%t</gsx:contactok>
                   <gsx:email>%s</gsx:email>
                   <gsx:phone>%s</gsx:phone>
                   <gsx:created>%s</gsx:created>
               </entry>
        `
	return fmt.Sprintf(template, attrs.Id, attrs.Location, attrs.Lat, attrs.Lng, attrs.Image, attrs.Story, attrs.ContactOk, attrs.Email, attrs.Phone, attrs.Created)
}

func push(filename string) error {
	h := hoops.ContributedHoop{}
	hoop := hoops.Hoop(&h)
	reader := hoops.FilesystemHoopReader{DataDir: conf.DataDir}
	reader.ReadFromFile(&hoop, filename)

	client, err := getClient()
	urls, err := getWorksheetUrls(client, 0)
	addRowURL, err := getAddRowURL(urls)

	row := rowXml(hoop)
	_, err = post(client, addRowURL, "application/atom+xml", row)

	if err != nil {
		return err
	}

	return nil
}

func main() {
	flag.Parse()

	if configFilename == "" {
		fmt.Printf("You must specify a configuration file\n")
		os.Exit(1)
	}
	err := hoops.ParseConfig(configFilename, &conf)
	if err != nil {
		log.Fatal(err)
	}

	args := flag.Args()
	cmd := args[0]

	switch cmd {
	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		os.Exit(1)
	case "oauthtoken":
		getOAuthToken()
	case "push":
		err = push(args[1])
	}

	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
}
