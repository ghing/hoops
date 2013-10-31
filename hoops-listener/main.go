package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"mime"
	"net/http"
	"net/smtp"

	"github.com/ghing/hoops"
)

var conf hoops.HoopsConfig
var configFilename string
var sendEmail bool = false
var smtpAuth smtp.Auth

func sendNotificationEmail(auth smtp.Auth, h *hoops.ContributedHoop) {
	// TODO: More informative message
	body := "From: " + conf.EmailSendingEmail + "\r\n" +
		"To: " + conf.NotificationEmail + "\r\n" +
		"Subject: " + "[hoops] New hoop added" + "\r\n\r\n" +
		"A new hoop has been added. It's ID is " + h.Id()
	err := smtp.SendMail(
		conf.EmailSendingHost+":25",
		auth,
		conf.EmailSendingEmail,
		[]string{conf.NotificationEmail},
		[]byte(body),
	)
	if err != nil {
		log.Println(err)
	}
}

func hoopsHandler(w http.ResponseWriter, r *http.Request) {
	var hoopJSON []byte
	v := r.Header.Get("Content-Type")
	d, _, err := mime.ParseMediaType(v)
	if err != nil || d != "multipart/form-data" {
		// TODO: Accept JSON
		http.Error(w, "Request must be encoded as multipart/form-data", http.StatusBadRequest)
		return
	}

	saver := hoops.FilesystemHoopSaver{DataDir: conf.DataDir}
	mediaSaver := hoops.FilesystemHoopMediaSaver{DataDir: conf.DataDir}
	hoop := hoops.NewContributedHoop()
	hoop.FromRequest(r)
	err = hoop.Save(hoops.HoopSaver(saver), hoops.HoopMediaSaver(mediaSaver))
	if err != nil {
		http.Error(w, "Error saving hoop", http.StatusInternalServerError)
		return
	}
	if sendEmail {
		go sendNotificationEmail(smtpAuth, hoop)
	}
	w.Header().Set("Content-Type", "application/json")
	hoopJSON, err = json.Marshal(hoop)
	fmt.Fprintf(w, string(hoopJSON))
}

func init() {
	conf = hoops.HoopsConfig{}
	flag.StringVar(&configFilename, "config", "", "Configuration file")
}

func main() {
	flag.Parse()
	if configFilename == "" {
		log.Fatal("You must specify a configuration file")
	}
	err := hoops.ParseConfig(configFilename, &conf)
	if err != nil {
		log.Fatal(err)
	}
	if conf.EmailSendingUsername != "" && conf.EmailSendingPassword != "" && conf.EmailSendingHost != "" {
		sendEmail = true
		smtpAuth = smtp.PlainAuth(
			"",
			conf.EmailSendingUsername,
			conf.EmailSendingPassword,
			conf.EmailSendingHost,
		)
	}

	http.HandleFunc("/api/0.1/hoops/", hoopsHandler)
	http.ListenAndServe(fmt.Sprintf(":%d", conf.Port), nil)
}
