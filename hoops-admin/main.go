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
var cmd Command

type Command interface {
	FlagSet() *flag.FlagSet
	Handle() error
	SetConf(hoops.HoopsConfig)
}

type BaseCommand struct {
	conf  hoops.HoopsConfig
	flags *flag.FlagSet
}

func (c *BaseCommand) SetConf(conf hoops.HoopsConfig) {
	c.conf = conf
}

type GetOAuthToken struct {
	BaseCommand
	code *string
}

func (c *GetOAuthToken) FlagSet() *flag.FlagSet {
	c.flags = flag.NewFlagSet("oauthtoken", flag.ExitOnError)
	c.code = c.flags.String("code", "", "OAuth 2 Authorization Code")

	return c.flags
}

func (c *GetOAuthToken) Handle() error {
	// Set up a configuration.
	oauthConfig := gspreadsheets.GetOAuthConfig(c.conf.OAuthClientId, c.conf.OAuthClientSecret, c.conf.OAuthTokenCacheFile)

	if *c.code == "" {
		url := oauthConfig.AuthCodeURL("")
		fmt.Println("Visit this URL to get a code, then run again with -code=YOUR_CODE\n")
		fmt.Println(url)
		return nil
	}

	// Set up a Transport using the config.
	transport := &oauth.Transport{Config: oauthConfig}

	_, err := transport.Exchange(*c.code)
	if err != nil {
		log.Fatal("Exchange:", err)
	}

	// (The Exchange method will automatically cache the token.)
	fmt.Printf("Token is cached in %v\n", oauthConfig.TokenCache)

	return nil
}

type Push struct {
	BaseCommand
}

func (c *Push) FlagSet() *flag.FlagSet {
	c.flags = flag.NewFlagSet("push", flag.ExitOnError)

	return c.flags
}

func (c *Push) Handle() error {
	filename := c.flags.Arg(0)
	h := hoops.ContributedHoop{}
	hoop := hoops.Hoop(&h)
	reader := hoops.FilesystemHoopReader{DataDir: conf.DataDir}
	saver := hoops.GoogleSpreadsheetHoopSaver{
		Key:                 c.conf.SpreadsheetKey,
		OAuthClientId:       c.conf.OAuthClientId,
		OAuthClientSecret:   c.conf.OAuthClientSecret,
		OAuthTokenCacheFile: c.conf.OAuthTokenCacheFile,
	}
	reader.ReadFromFile(&hoop, filename)

	err := hoop.Save(saver)

	if err != nil {
		return err
	}

	return nil
}

type Show struct {
	BaseCommand
}

func (c *Show) Handle() error {
	filename := c.flags.Arg(0)
	h := &hoops.ContributedHoop{}
	hoop := hoops.Hoop(h)
	reader := hoops.FilesystemHoopReader{DataDir: c.conf.DataDir}
	reader.ReadFromFile(&hoop, filename)

	fmt.Print(h)

	return nil
}

func (c *Show) FlagSet() *flag.FlagSet {
	c.flags = flag.NewFlagSet("show", flag.ExitOnError)

	return c.flags
}

func init() {
	conf = hoops.HoopsConfig{}
}

func main() {
	cmdName := os.Args[1]

	switch cmdName {
	default:
		fmt.Printf("Unknown command: %s\n", cmdName)
		os.Exit(1)
	case "oauthtoken":
		cmd = &GetOAuthToken{}
	case "push":
		cmd = &Push{}
	case "show":
		cmd = &Show{}
	}

	flags := cmd.FlagSet()
	flags.StringVar(&configFilename, "config", "", "Configuration file")
	flags.Parse(os.Args[2:])
	if configFilename == "" {
		fmt.Printf("You must specify a configuration file\n")
		os.Exit(1)
	}
	err := hoops.ParseConfig(configFilename, &conf)
	if err != nil {
		log.Fatal(err)
	}
	cmd.SetConf(conf)

	err = cmd.Handle()

	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
}
