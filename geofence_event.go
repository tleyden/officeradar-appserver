package officeradar

type GeofenceEvent struct {
	OfficeRadarDoc
	Action    string `json:"action"`
	BeaconId  string `json:"beacon"`
	CreatedAt string `json:"created_at"`
	ProfileId string `json:"profile"`
}

func (e GeofenceEvent) ActionPastTense() string {

	switch e.Action {
	case "entry":
		return "entered"
	case "exit":
		return "exited"
	}
	return "error"
}
