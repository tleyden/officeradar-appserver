package officeradar

import "github.com/tleyden/go-couch"

type Beacon struct {
	OfficeRadarDoc
	Desc         string `json:"desc"`
	Location     string `json:"location"`
	Uuid         string `json:"uuid"`
	Major        int    `json:"major"`
	Minor        int    `json:"minor"`
	Organization string `json:"organization"`
}

func FetchBeacon(db couch.Database, beaconId string) (*Beacon, error) {

	beaconDoc := Beacon{}
	err := db.Retrieve(beaconId, &beaconDoc)
	if err != nil {
		return nil, err
	}

	return &beaconDoc, nil

}
