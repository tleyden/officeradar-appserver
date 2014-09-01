package officeradar

import "time"

/*
Alert examples
  - Next time I am in range of BeaconX
    - Action: send me customized message
  - Next time Jens and I in same beacon
    - Condition: 1 hour time window
    - Action: send message to me, or
    - Action: send message to both Jens and I
  - Next time Jens is in range of BeaconX
    - Action: send message to me, or
    - Action: send customized message Jens
  - Next time anyone in org enters any beacon after 2 week absence
    - Action: send message to list of users
    - Note: this is persistent/ongoing
*/

// A geofence alert triggered if any of the users enters within range of a specific beacon.
// Useful for things like leaving notes at a beacon for anyone who enters the region,
// Eg, "Can the first person into the office clean up the happy hour messs?"
type AnyUsersPresentAlert struct {
	BaseAlert
	Users  []OfficeRadarProfile // if any of these users are present, alert can fire.
	Beacon Beacon               // the beacon of interest
}

func NewAnyUsersPresentAlert() *AnyUsersPresentAlert {
	alert := &AnyUsersPresentAlert{}
	alert.Type = "any_users_present_alert"
	return alert
}

func (a *AnyUsersPresentAlert) Process(e GeofenceEvent) (bool, error) {
	return true, nil
}

// A geofence alert triggered if all of the users enter within range of one the
// beacons in the list of beacons, within the specified time window.
// Eg, "Send me an alert when Jens and I are in the same office within 1/2 hour of eachother"
type AllUsersPresentAlert struct {
	BaseAlert
	Users  []OfficeRadarProfile // users who must be in range of beacon, within time window
	Window time.Duration        // max time window for user appearances of multi-user alerts
	Beacon []Beacon             // the beacons of interest
	State  map[string]time.Time // which users have been seen so far
}

// A geofence alert triggered if any of the specified users enter any of the specified
// beacons after not having been seen at that beacon since minLastSeenAgo time duration.
type SurpriseAppearanceAlert struct {
	BaseAlert
	Users          []OfficeRadarProfile // users for which this alert can fire
	Beacon         []Beacon             // beacons for which this alert can fire
	MinLastSeenAgo time.Duration        // user(s) must not seen at beacon for time duration
}

// The base geofence alert that contains fields used in all types of geofence alerts
type BaseAlert struct {
	OfficeRadarDoc
	Actions         []AlertAction // the actions to be performed when alert triggers
	Sticky          bool          // should this alert remain after it fires?
	ReactivateAfter time.Duration // delay before reaactivating a sticky alert
	ActiveOn        time.Time     // the time after which this alert becomes active
}

type Alerter interface {
	Process(geofenceEvent GeofenceEvent) (bool, error)
}

type AlertAction struct {
	Recipient OfficeRadarProfile // the user that will receive a message
	Message   string             // the message to be sent
}

type AlertHandler func([]AlertAction)
