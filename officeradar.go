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

func (o OfficeRadarApp) FollowChangesFeed(startingSince string) {

	handleChange := func(reader io.Reader) interface{} {
		logg.LogTo("OFFICERADAR", "handleChange() callback called")
		changes, err := decodeChanges(reader)
		if err != nil {
			logg.LogTo("OFFICERADAR", "error decoding changes: %v", err)
			return nil // stop changes feed
		}

		logg.LogTo("OFFICERADAR", "changes: %v", changes)

		o.processChanges(changes)

		since := changes.LastSequence
		logg.LogTo("OFFICERADAR", "returning since: %v", since)

		return since

	}

	options := map[string]interface{}{}
	if startingSince != "" {
		logg.LogTo("OFFICERADAR", "startingSince not empty: %v", startingSince)
		options["since"] = startingSince
	} else {
		// find the sequence of most recent change
		lastSequence, err := o.Database.LastSequence()
		if err != nil {
			logg.LogPanic("Error getting LastSequence: %v", err)
			return
		}
		options["since"] = lastSequence
	}

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
	o.sendPushToSubscriber(profileDoc.Id, "Hello")

}

func (o OfficeRadarApp) processChangedGeofenceEvent(change couch.Change) {

	geofenceDoc := GeofenceEvent{}
	err := o.Database.Retrieve(change.Id, &geofenceDoc)
	if err != nil {
		errMsg := fmt.Errorf("Load fail: %v - %v", change.Id, err)
		logg.LogError(errMsg)
		return
	}

	o.noisyTempAlert(geofenceDoc)

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

		result, err := alert.Process(geofenceEvent)
		if err != nil {
			// TODO: handle failed alert by logging it and
			// deleting the alert
			continue
		}

		logg.LogTo("OFFICERADAR", "alert result: %v", result)

		// alert.InvokeA

		// if true, then invoke actions associated with alert

		// if it's sticky, then:
		// create a new alert based on previous alert, (but don't copy over state)
		// set the activeOn time to current time + reactivateAfter

		// delete the alert

	}

}

// Use a view query to find all active alerts
func (o OfficeRadarApp) findActiveAlerts() ([]Alerter, error) {
	// TODO: query view via go-couchdb
	// Based on type field, instantiate correct alert instance from doc json
	return []Alerter{}, nil

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
