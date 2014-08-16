package officeradar

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/couchbaselabs/logg"
	"github.com/tleyden/go-couch"
)

type OfficeRadarApp struct {
	DatabaseURL string
	UniqushURL  string
	Database    couch.Database
}

type OfficeRadarDoc struct {
	Revision string `json:"_rev"`
	Id       string `json:"_id"`
	Type     string `json:"type"`
}

type OfficeRadarProfile struct {
	OfficeRadarDoc
	DeviceTokens []string `json:"deviceTokens"`
	Name         string   `json:"name"`
	AuthSystem   string   `json:"authSystem"`
}

type stringmap map[string]interface{}

const (
	UNIQUSH_OFFICERADAR_SERVICE = "officeradar"
)

func NewOfficeRadarApp(databaseURL string, uniqushURL string) *OfficeRadarApp {
	return &OfficeRadarApp{
		DatabaseURL: databaseURL,
		UniqushURL:  uniqushURL,
	}
}

func (o *OfficeRadarApp) InitApp() error {
	db, err := couch.Connect(o.DatabaseURL)
	if err != nil {
		logg.LogPanic("Error connecting to db: %v", err)
		return err
	}
	o.Database = db
	return nil
}

func (o OfficeRadarApp) FollowChangesFeed(since interface{}) {

	handleChange := func(reader io.Reader) interface{} {
		logg.LogTo("OFFICERADAR", "handleChange() callback called")
		changes, err := decodeChanges(reader)
		if err == nil {
			logg.LogTo("OFFICERADAR", "changes: %v", changes)

			o.processChanges(changes)

			since = changes.LastSequence

		} else {
			logg.LogTo("OFFICERADAR", "error decoding changes: %v", err)

		}

		logg.LogTo("OFFICERADAR", "returning since: %v", since)
		return since

	}

	options := stringmap{"since": since}
	options["feed"] = "longpoll"
	o.Database.Changes(handleChange, options)

}

func (o OfficeRadarApp) processChanges(changes couch.Changes) {

	for _, change := range changes.Results {
		logg.LogTo("OFFICERADAR", "change: %v", change)

		if change.Deleted {
			logg.LogTo("OFFICERADAR", "change was deleted, skipping")
			continue
		}

		doc := OfficeRadarDoc{}
		err := o.Database.Retrieve(change.Id, &doc)
		if err != nil {
			errMsg := fmt.Errorf("Didn't retrieve: %v - %v", change.Id, err)
			logg.LogError(errMsg)
			continue
		}

		logg.LogTo("OFFICERADAR", "doc: %+v", doc)

		switch doc.Type {
		case "profile":
			o.processChangedProfile(change)
		}

	}

}

func (o OfficeRadarApp) processChangedProfile(change couch.Change) {

	profileDoc := OfficeRadarProfile{}
	err := o.Database.Retrieve(change.Id, &profileDoc)
	if err != nil {
		errMsg := fmt.Errorf("Load fail: %v - %v", change.Id, err)
		logg.LogError(errMsg)
		return
	}
	logg.LogTo("OFFICERADAR", "profileDoc: %+v", profileDoc)

	o.registerDeviceTokens(profileDoc)
	o.sendPushToSubscriber(profileDoc, "Hello")

}

func (o OfficeRadarApp) registerDeviceTokens(profileDoc OfficeRadarProfile) {

	endpointUrl := fmt.Sprintf("%s/subscribe", o.UniqushURL)

	for _, deviceToken := range profileDoc.DeviceTokens {

		formValues := url.Values{
			"service":         {UNIQUSH_OFFICERADAR_SERVICE},
			"subscriber":      {profileDoc.Id},
			"pushservicetype": {"apns"},
			"devtoken":        {deviceToken},
		}
		logg.LogTo("OFFICERADAR", "post to %v with vals: %v", endpointUrl, formValues)

		resp, err := http.PostForm(endpointUrl, formValues)
		if err != nil {
			errMsg := fmt.Errorf("Failed to add uniqush subscriber: %v - %v", profileDoc, err)
			logg.LogError(errMsg)
			continue
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			errMsg := fmt.Errorf("Failed to read body: %v - %v", profileDoc, err)
			logg.LogError(errMsg)
			continue
		}

		logg.LogTo("OFFICERADAR", "uniqush response body: %v", string(body))

	}

}

func (o OfficeRadarApp) sendPushToSubscriber(profileDoc OfficeRadarProfile, msg string) {

	endpointUrl := fmt.Sprintf("%s/push", o.UniqushURL)
	formValues := url.Values{
		"service":    {UNIQUSH_OFFICERADAR_SERVICE},
		"subscriber": {profileDoc.Id},
		"msg":        {msg},
	}
	logg.LogTo("OFFICERADAR", "post to %v with vals: %v", endpointUrl, formValues)

	resp, err := http.PostForm(endpointUrl, formValues)
	if err != nil {
		errMsg := fmt.Errorf("Failed to send push: %v - %v", profileDoc, err)
		logg.LogError(errMsg)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		errMsg := fmt.Errorf("Failed to read body: %v - %v", profileDoc, err)
		logg.LogError(errMsg)
	}
	logg.LogTo("OFFICERADAR", "uniqush response body: %v", string(body))

}

func decodeChanges(reader io.Reader) (couch.Changes, error) {

	changes := couch.Changes{}
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(&changes)
	if err != nil {
		logg.LogTo("OFFICERADAR", "Err decoding changes: %v", err)
	}
	return changes, err

}
