package officeradar

import "time"

const (
	ACTION_ENTRY = "entry"
	ACTION_EXIT  = "exit"
)

type GeofenceEvent struct {
	OfficeRadarDoc
	Action    string `json:"action"`
	BeaconId  string `json:"beacon"`
	CreatedAt string `json:"created_at"` // eg, "2014-08-29T01:19:15.388Z" - RFC3339
	ProfileId string `json:"profile"`
}

func (e GeofenceEvent) ActionPastTense() string {

	switch e.Action {
	case ACTION_ENTRY:
		return "entered"
	case ACTION_EXIT:
		return "exited"
	}
	return "error"
}

func (e GeofenceEvent) CreatedAtTime() (time.Time, error) {

	return time.Parse(time.RFC3339, e.CreatedAt)

}
