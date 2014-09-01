package officeradar

import (
	"testing"

	"github.com/couchbaselabs/go.assert"
	"github.com/couchbaselabs/logg"
)

func init() {
	logg.LogKeys["TEST"] = true

}

func TestAnyUsersPresentAlert(t *testing.T) {

	alert := NewAnyUsersPresentAlert()
	foo := OfficeRadarProfile{OfficeRadarDoc: OfficeRadarDoc{Id: "foo"}}
	bar := OfficeRadarProfile{OfficeRadarDoc: OfficeRadarDoc{Id: "bar"}}

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
