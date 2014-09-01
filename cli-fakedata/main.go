package main

import (
	"github.com/alecthomas/kingpin"
	"github.com/couchbaselabs/logg"
	"github.com/tleyden/go-couch"
	"github.com/tleyden/officeradar-appserver"
)

var (
	sgUrlDescription = "Sync gateway url, with db name and no trailing slash"
	sgUrl            = kingpin.Arg("sg-url", sgUrlDescription).Required().String()
)

func init() {
	logg.LogKeys["CLI"] = true
	logg.LogKeys["OFFICERADAR"] = true
}

func main() {
	kingpin.Parse()
	if *sgUrl == "" {
		kingpin.UsageErrorf("sgURL is empty")
		return
	}

	db, err := couch.Connect(*sgUrl)
	if err != nil {
		logg.LogPanic("Error connecting to db: %v", err)
		return
	}

	createFakeData(db)

}

func createFakeData(db couch.Database) {

	sfBeaconId := "sfBeaconId" // replace w/ real id
	mvBeaconId := "mvBeaconId"
	jensId := "jensId"
	traunsId := "traunsId"
	geofenceId := "geofenceId"

	sfBeacon := officeradar.Beacon{
		OfficeRadarDoc: officeradar.OfficeRadarDoc{Id: sfBeaconId, Type: "beacon"},
		Desc:           "sf beacon",
	}
	_, _, err := db.Insert(sfBeacon)
	if err != nil {
		logg.LogPanic("Could not insert beacon: %v", err)
	}

	mvBeacon := officeradar.Beacon{
		OfficeRadarDoc: officeradar.OfficeRadarDoc{Id: mvBeaconId, Type: "beacon"},
		Desc:           "mv beacon",
	}
	_, _, err = db.Insert(mvBeacon)
	if err != nil {
		logg.LogPanic("Could not insert beacon: %v", err)
	}

	jensProfile := officeradar.OfficeRadarProfile{
		OfficeRadarDoc: officeradar.OfficeRadarDoc{Id: jensId, Type: "profile"},
	}
	_, _, err = db.Insert(jensProfile)
	if err != nil {
		logg.LogPanic("Could not insert profile: %v", err)
	}

	traunsProfile := officeradar.OfficeRadarProfile{
		OfficeRadarDoc: officeradar.OfficeRadarDoc{Id: traunsId, Type: "profile"},
	}
	_, _, err = db.Insert(traunsProfile)
	if err != nil {
		logg.LogPanic("Could not insert profile: %v", err)
	}

	geofenceEvent := officeradar.GeofenceEvent{
		OfficeRadarDoc: officeradar.OfficeRadarDoc{Id: geofenceId, Type: "geofence_event"},
		BeaconId:       sfBeacon.Id,
		ProfileId:      jensProfile.Id,
	}
	_, _, err = db.Insert(geofenceEvent)
	if err != nil {
		logg.LogPanic("Could not insert to db: %v", err)
	}

}
