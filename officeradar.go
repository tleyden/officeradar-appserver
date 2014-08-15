package officeradar

import (
	"encoding/json"
	"fmt"
	"io"

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
		if doc.Type == "profile" {

			profileDoc := OfficeRadarProfile{}
			err := o.Database.Retrieve(change.Id, &profileDoc)
			if err != nil {
				errMsg := fmt.Errorf("Load fail: %v - %v", change.Id, err)
				logg.LogError(errMsg)
				continue
			}
			logg.LogTo("OFFICERADAR", "profileDoc: %+v", profileDoc)

			// register this user with Uniqush

		}

	}

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
