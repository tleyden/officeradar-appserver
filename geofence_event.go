package officeradar

const (
	ACTION_ENTRY = "entry"
	ACTION_EXIT  = "exit"
)

type GeofenceEvent struct {
	OfficeRadarDoc
	Action    string `json:"action"`
	BeaconId  string `json:"beacon"`
	CreatedAt string `json:"created_at"`
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
