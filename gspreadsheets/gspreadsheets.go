package gspreadsheets

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"code.google.com/p/goauth2/oauth"
)

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

type Spreadsheet struct {
	Key    string
	Client *http.Client
}

type Worksheet struct {
	PostUrl string
	Client  *http.Client
}

func GetOAuthConfig(clientId string, clientSecret string, tokenCacheFile string) *oauth.Config {
	return &oauth.Config{
		ClientId:     clientId,
		ClientSecret: clientSecret,
		RedirectURL:  "http://apps.terrorware.com/hoops/",
		Scope:        "https://spreadsheets.google.com/feeds https://docs.google.com/feeds",
		AuthURL:      "https://accounts.google.com/o/oauth2/auth",
		TokenURL:     "https://accounts.google.com/o/oauth2/token",
		TokenCache:   oauth.CacheFile(tokenCacheFile),
		AccessType:   "offline",
	}
}

func GetClient(clientId string, clientSecret string, tokenCacheFile string) (*http.Client, error) {
	oauthConfig := GetOAuthConfig(clientId, clientSecret, tokenCacheFile)
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

func (s Spreadsheet) getWorksheetUrls(worksheetIndex int) (map[string]string, error) {
	feed := Feed{}
	requestURL := fmt.Sprintf("https://spreadsheets.google.com/feeds/worksheets/%s/private/full", s.Key)
	err := getFeed(s.Client, requestURL, &feed)

	if err != nil {
		return nil, errors.New("Request failed")
	}

	entry := feed.Entry[worksheetIndex]
	urls := getUrls(entry.Link)

	listFeedURL, err := getListFeedURL(urls)
	err = getFeed(s.Client, listFeedURL, &feed)
	if err != nil {
		return nil, err
	}

	return getUrls(feed.Link), nil
}

func (s Spreadsheet) GetWorksheet(worksheetIndex int) (*Worksheet, error) {
	urls, err := s.getWorksheetUrls(worksheetIndex)
	if err != nil {
		return nil, err
	}

	addRowURL, err := getAddRowURL(urls)

	w := &Worksheet{
		Client:  s.Client,
		PostUrl: addRowURL,
	}

	return w, nil
}

func (w Worksheet) AddRow(rowXml string) error {
	_, err := post(w.Client, w.PostUrl, "application/atom+xml", rowXml)

	if err != nil {
		return err
	}

	return nil
}
