package officeradar

import "github.com/tleyden/go-couch"

type OfficeRadarProfile struct {
	OfficeRadarDoc
	DeviceTokens []string `json:"deviceTokens"`
	Name         string   `json:"name"`
	AuthSystem   string   `json:"authSystem"`
}

func FetchOfficeRadarProfile(db couch.Database, profileId string) (*OfficeRadarProfile, error) {

	profileDoc := OfficeRadarProfile{}
	err := db.Retrieve(profileId, &profileDoc)
	if err != nil {
		return nil, err
	}

	return &profileDoc, nil

}
