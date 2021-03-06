package officeradar

import (
	"testing"
	"time"

	"github.com/couchbaselabs/go.assert"
	"github.com/couchbaselabs/logg"
)

func init() {
	logg.LogKeys["TEST"] = true
	logg.LogKeys["OFFICERADAR"] = true

}

func TestAnyUsersPresentAlert(t *testing.T) {

	alert := NewAnyUsersPresentAlert()

	foo := OfficeRadarProfile{OfficeRadarDoc: OfficeRadarDoc{Id: "foo"}}
	bar := OfficeRadarProfile{OfficeRadarDoc: OfficeRadarDoc{Id: "bar"}}

	action := AlertAction{
		Recipient: foo.Id,
		Message:   "yo",
	}
	alert.Actions = []AlertAction{action}

	beacon := Beacon{
		OfficeRadarDoc: OfficeRadarDoc{Id: "fake_beacon_id"},
	}

	alert.Users = []OfficeRadarProfile{foo, bar}
	alert.Beacon = beacon

	geofenceEvent := GeofenceEvent{
		Action:    ACTION_ENTRY,
		BeaconId:  beacon.Id,
		ProfileId: foo.Id,
	}

	fired, error := alert.Process(geofenceEvent)
	assert.True(t, error == nil)
	assert.True(t, fired)

	// try with a user that shouldn't match
	geofenceEvent.ProfileId = "unknown_user"
	fired2, error := alert.Process(geofenceEvent)
	assert.True(t, error == nil)
	assert.False(t, fired2)

}

func TestSurpriseAppearanceAlert(t *testing.T) {

	alert := NewSurpriseAppearanceAlert()

	foo := OfficeRadarProfile{OfficeRadarDoc: OfficeRadarDoc{Id: "foo"}}
	bar := OfficeRadarProfile{OfficeRadarDoc: OfficeRadarDoc{Id: "bar"}}
	alert.Users = []OfficeRadarProfile{foo, bar}

	beacon1 := Beacon{
		OfficeRadarDoc: OfficeRadarDoc{Id: "fake_beacon_id1"},
	}
	beacon2 := Beacon{
		OfficeRadarDoc: OfficeRadarDoc{Id: "fake_beacon_id2"},
	}
	alert.Beacons = []Beacon{beacon1, beacon2}

	// only fire as long as we haven't seen user for two weeks
	alert.MinLastSeenAgo = (14 * 24 * time.Hour)

	// set a fake last seen func that says the user was just
	// recently seen
	alert.LastSeenFunc = func(profileId, beaconId string) (bool, time.Time) {
		return true, time.Now()
	}

	// create a geofence event that happened just now for
	// the initial user and beacon
	createdAtNow := time.Now().Format(time.RFC3339)
	geofenceEvent := GeofenceEvent{
		Action:    ACTION_ENTRY,
		BeaconId:  beacon1.Id,
		ProfileId: foo.Id,
		CreatedAt: createdAtNow,
	}

	// the alert should not fire
	fired, error := alert.Process(geofenceEvent)
	assert.True(t, error == nil)
	assert.False(t, fired)

	// set a fake last seen func that says the user hasn't been
	// seen for three weeks
	alert.LastSeenFunc = func(profileId, beaconId string) (bool, time.Time) {
		return true, time.Now().Add(-1 * 21 * 24 * time.Hour)
	}

	// send in geofence event again, the alert should fire this time
	fired2, error := alert.Process(geofenceEvent)
	assert.True(t, error == nil)
	assert.True(t, fired2)

}

func TestAllUsersPresentAlert(t *testing.T) {

	alert := NewAllUsersPresentAlert()

	alert.Window = time.Hour

	foo := OfficeRadarProfile{OfficeRadarDoc: OfficeRadarDoc{Id: "foo"}}
	bar := OfficeRadarProfile{OfficeRadarDoc: OfficeRadarDoc{Id: "bar"}}
	alert.Users = []OfficeRadarProfile{foo, bar}

	beacon1 := Beacon{
		OfficeRadarDoc: OfficeRadarDoc{Id: "fake_beacon_id1"},
	}
	beacon2 := Beacon{
		OfficeRadarDoc: OfficeRadarDoc{Id: "fake_beacon_id2"},
	}
	alert.Beacons = []Beacon{beacon1, beacon2}

	// foo hasn't been seen in two weeks
	// bar just seen recently
	alert.LastSeenFunc = func(profileId, beaconId string) (bool, time.Time) {
		switch profileId {
		case foo.Id:
			return true, time.Now().Add(-1 * 21 * 24 * time.Hour)
		case bar.Id:
			return true, time.Now()
		}
		return false, time.Now()
	}

	createdAtNow := time.Now().Format(time.RFC3339)
	geofenceEvent := GeofenceEvent{
		Action:    ACTION_ENTRY,
		BeaconId:  beacon1.Id,
		ProfileId: bar.Id,
		CreatedAt: createdAtNow,
	}

	// the alert should not fire
	fired, error := alert.Process(geofenceEvent)
	assert.True(t, error == nil)
	assert.False(t, fired)

	// foo and bar both just seen recently
	alert.LastSeenFunc = func(profileId, beaconId string) (bool, time.Time) {
		switch profileId {
		case foo.Id:
			return true, time.Now().Add(-1 * 2 * time.Minute)
		case bar.Id:
			return true, time.Now()
		}
		return false, time.Now()
	}

	// send in geofence event again, the alert should  fire
	fired2, error := alert.Process(geofenceEvent)
	assert.True(t, error == nil)
	assert.True(t, fired2)

}
