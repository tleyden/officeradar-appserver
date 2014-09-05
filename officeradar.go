package officeradar

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/couchbaselabs/logg"
	"github.com/tleyden/go-couch"
)

type OfficeRadarApp struct {
	DatabaseURL string
	UniqushURL  string
	Database    couch.Database
}

type OfficeRadarDoc struct {
	database couch.Database
	Revision string `json:"_rev"`
	Id       string `json:"_id"`
	Type     string `json:"type"`
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

func (o *OfficeRadarApp) InitHardcodedAlerts() error {

	db := o.Database

	alert := NewAnyUsersPresentAlert()

	// it's necessary to set an id, because otherwise it will default to
	// empty string id (as opposed to not including id in json), which
	// causes sync gateway to return an error (technically a 301)
	alert.Id = "hardcoded_alert_1"

	retrievedAlert := &AnyUsersPresentAlert{}
	err := db.Retrieve(alert.Id, retrievedAlert)
	if err == nil {
		logg.LogTo("OFFICERADAR", "already have alert, skip adding alert")
		return nil
	}

	sfBeaconId := "df7172f4e29b4d10881229810b9af710"
	mvBeaconId := "4a83813db6ce76e9618793cf483cfa10"
	macBeaconId := "b18b572cb8a4ea6d5ce12b4620c7b90f"
	jensId := "242941625916974"
	traunsId := "727846993927551"

	sfBeacon := Beacon{}
	err = db.Retrieve(sfBeaconId, &sfBeacon)
	if err != nil {
		logg.LogPanic("Could not find beacon: %v", err)
	}

	mvBeacon := Beacon{}
	err = db.Retrieve(mvBeaconId, &mvBeacon)
	if err != nil {
		logg.LogPanic("Could not find beacon: %v", err)
	}
	logg.LogTo("OFFICERADAR", "mvBeacon: %v", mvBeacon)

	macBeacon := Beacon{}
	err = db.Retrieve(macBeaconId, &macBeacon)
	if err != nil {
		logg.LogPanic("Could not find beacon: %v", err)
	}
	logg.LogTo("OFFICERADAR", "macBeacon: %+v", macBeacon)

	jensProfile := OfficeRadarProfile{}
	err = db.Retrieve(jensId, &jensProfile)
	if err != nil {
		logg.LogPanic("Could not find profile: %v", err)
	}

	traunsProfile := OfficeRadarProfile{}
	err = db.Retrieve(traunsId, &traunsProfile)
	if err != nil {
		logg.LogPanic("Could not find profile: %v", err)
	}

	alert.Users = []OfficeRadarProfile{jensProfile, traunsProfile}
	alert.Beacon = sfBeacon

	action := AlertAction{
		Recipient: "727846993927551",
		Message:   "Jens or Traun passed by a beacon",
	}
	alert.Actions = []AlertAction{action}
	alert.Sticky = true
	alert.ReactivateAfter = time.Second * 30

	id, rev, err := db.Insert(alert)
	if err != nil {
		logg.LogPanic("Could not create alert: %v", err)
	}

	logg.LogTo("OFFICERADAR", "created alert id: %v rev: %v", id, rev)
	return nil

}

func (o OfficeRadarApp) FollowChangesFeed(startingSince string) {

	var since interface{}

	handleChange := func(reader io.Reader) interface{} {
		logg.LogTo("OFFICERADAR", "handleChange() callback called")
		changes, err := decodeChanges(reader)
		if err != nil {
			// it's very common for this to timeout while waiting for new changes.
			// since we want to follow the changes feed forever, just log an error
			// TODO: don't even log an error if its an io.Timeout, just noise
			logg.LogTo("OFFICERADAR", "%T decoding changes: %v.", err, err)
			return since
		}

		logg.LogTo("OFFICERADAR", "changes: %v", changes)

		o.processChanges(changes)

		since = changes.LastSequence
		logg.LogTo("OFFICERADAR", "returning since: %v", since)

		return since

	}

	options := map[string]interface{}{}
	if startingSince != "" {
		logg.LogTo("OFFICERADAR", "startingSince not empty: %v", startingSince)
		since = startingSince
	} else {
		// find the sequence of most recent change
		lastSequence, err := o.Database.LastSequence()
		if err != nil {
			logg.LogPanic("Error getting LastSequence: %v", err)
			return
		}
		since = lastSequence
	}

	options["since"] = since
	options["feed"] = "longpoll"
	logg.LogTo("OFFICERADAR", "Following changes feed: %+v", options)
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
		case "geofence_event":
			o.processChangedGeofenceEvent(change)
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

}

func (o OfficeRadarApp) processChangedGeofenceEvent(change couch.Change) {

	geofenceDoc := GeofenceEvent{}
	err := o.Database.Retrieve(change.Id, &geofenceDoc)
	if err != nil {
		errMsg := fmt.Errorf("Load fail: %v - %v", change.Id, err)
		logg.LogError(errMsg)
		return
	}

	// o.noisyTempAlert(geofenceDoc)

	o.triggerAlerts(geofenceDoc)

}

func (o OfficeRadarApp) triggerAlerts(geofenceEvent GeofenceEvent) {

	activeAlerts, err := o.findActiveAlerts()
	if err != nil {
		errMsg := fmt.Errorf("Failed to find active alerts: %v", err)
		logg.LogError(errMsg)
		return
	}

	for _, alert := range activeAlerts {

		shouldFire, err := alert.Process(geofenceEvent)
		if err != nil {
			errMsg := fmt.Errorf("Alert failed to process event: %v", err)
			logg.LogError(errMsg)
			continue
		}

		logg.LogTo("OFFICERADAR", "alert.Process(): shouldFire = %v", shouldFire)

		if !shouldFire {
			continue
		}

		// invoke actions associated with alert
		o.invokeActions(alert, geofenceEvent)

		err = alert.RescheduleOrDelete()
		if err != nil {
			errMsg := fmt.Errorf("Unable to reschedule or delete alert %+v: err: %v", alert, err)
			logg.LogError(errMsg)
		}

	}

}

func (o OfficeRadarApp) invokeActions(alert Alerter, geofenceEvent GeofenceEvent) {

	defaultActionFunc := func(action AlertAction) error {
		logg.LogTo("OFFICERADAR", "invoke action on: %+v", action)
		o.sendPushToSubscriber(action.Recipient, action.Message)
		return nil
	}
	logg.LogTo("OFFICERADAR", "perform action: %v", defaultActionFunc)
	err := alert.PerformActions(defaultActionFunc)
	logg.LogTo("OFFICERADAR", "performed action: %v", defaultActionFunc)
	if err != nil {
		errMsg := fmt.Errorf("Alert failed to perform actions: %v", err)
		logg.LogError(errMsg)
		return
	}

}

// Use a view query to find all active alerts
func (o OfficeRadarApp) findActiveAlerts() ([]Alerter, error) {

	db := o.Database

	alerters := []Alerter{}
	alertIds := []string{"hardcoded_alert_1"}
	for _, alertId := range alertIds {
		retrievedAlert := &BaseAlert{}
		err := db.Retrieve(alertId, retrievedAlert)
		if err != nil {
			return []Alerter{}, err
		}
		switch retrievedAlert.Type {
		case DOC_TYPE_ANY_USERS_PRESENT_ALERT:
			alert := &AnyUsersPresentAlert{}
			alert.database = db
			err := db.Retrieve(alertId, alert)
			if err != nil {
				return []Alerter{}, err
			}
			alerters = append(alerters, alert)
		}
	}

	// TODO: query view via go-couchdb

	return alerters, nil

}

// This was added temporarily to test alerts.  This will get removed once
// the real alerts system is in place.
func (o OfficeRadarApp) noisyTempAlert(geofenceEvent GeofenceEvent) {

	// create the message for the alert
	msg := o.createAlertMessage(geofenceEvent)

	// send the alert to a hardcoded list of user id's (for now)
	recipients := []string{"727846993927551"}
	for _, recipient := range recipients {
		o.sendPushToSubscriber(recipient, msg)
	}

}

func (o OfficeRadarApp) createAlertMessage(geofenceEvent GeofenceEvent) string {

	// example message: "<name> entered|exited <location>"

	// find the name
	profileDoc, err := FetchOfficeRadarProfile(o.Database, geofenceEvent.ProfileId)
	if err != nil {
		errMsg := fmt.Errorf("Error loading profile from %+v: %v", geofenceEvent, err)
		logg.LogError(errMsg)
		return "Sorry, an error occurred"
	}

	// find the beacon
	beaconDoc, err := FetchBeacon(o.Database, geofenceEvent.BeaconId)
	if err != nil {
		errMsg := fmt.Errorf("Error loading beacon from %+v: %v", geofenceEvent, err)
		logg.LogError(errMsg)
		return "Sorry, an error occurred"
	}

	// get the action, eg, "entered" or "exited"
	action := geofenceEvent.ActionPastTense()

	msg := fmt.Sprintf("%s %s %s", profileDoc.Name, action, beaconDoc.Location)

	return msg

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

func (o OfficeRadarApp) sendPushToSubscriber(profileId string, msg string) {

	endpointUrl := fmt.Sprintf("%s/push", o.UniqushURL)
	formValues := url.Values{
		"service":    {UNIQUSH_OFFICERADAR_SERVICE},
		"subscriber": {profileId},
		"msg":        {msg},
	}
	logg.LogTo("OFFICERADAR", "post to %v with vals: %v", endpointUrl, formValues)

	resp, err := http.PostForm(endpointUrl, formValues)
	if err != nil {
		errMsg := fmt.Errorf("Failed to send push to: %v - %v", profileId, err)
		logg.LogError(errMsg)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		errMsg := fmt.Errorf("Failed to read body: %v - %v", profileId, err)
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
