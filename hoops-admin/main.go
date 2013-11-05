package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"code.google.com/p/goauth2/oauth"

	"github.com/ghing/hoops"
	"github.com/ghing/hoops/gspreadsheets"
)

var conf hoops.HoopsConfig
var configFilename string
var oauthCode string

func init() {
	conf = hoops.HoopsConfig{}
	flag.StringVar(&configFilename, "config", "", "Configuration file")
	flag.StringVar(&oauthCode, "code", "", "OAuth 2 Authorization Code")
}

func getOAuthToken() {
	// Set up a configuration.
	oauthConfig := gspreadsheets.GetOAuthConfig(conf.OAuthClientId, conf.OAuthClientSecret, conf.OAuthTokenCacheFile)

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

func push(filename string) error {
	h := hoops.ContributedHoop{}
	hoop := hoops.Hoop(&h)
	reader := hoops.FilesystemHoopReader{DataDir: conf.DataDir}
	saver := hoops.GoogleSpreadsheetHoopSaver{
		Key:                 conf.SpreadsheetKey,
		OAuthClientId:       conf.OAuthClientId,
		OAuthClientSecret:   conf.OAuthClientSecret,
		OAuthTokenCacheFile: conf.OAuthTokenCacheFile,
	}
	reader.ReadFromFile(&hoop, filename)

	err := hoop.Save(saver)

	if err != nil {
		return err
	}

	return nil
}

func show(filename string) error {
	h := &hoops.ContributedHoop{}
	hoop := hoops.Hoop(h)
	reader := hoops.FilesystemHoopReader{DataDir: conf.DataDir}
	reader.ReadFromFile(&hoop, filename)

	fmt.Print(h)

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
	case "show":
		err = show(args[1])
	}

	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
}
