package main

import (
	"github.com/alecthomas/kingpin"
	"github.com/couchbaselabs/logg"
	"github.com/tleyden/officeradar-appserver"
)

// This follows the changes feed of the OfficeRadar sync gateway database and:
//   - When a profile document is changed, it updates Uniqush with the subscribe/device token

var (
	sgUrlDescription = "Sync gateway url, with db name and no trailing slash"
	sgUrl            = kingpin.Arg("sg-url", sgUrlDescription).Required().String()
	uqUrlDescription = "Uniqush gateway url"
	uqUrl            = kingpin.Arg("uq-url", uqUrlDescription).Required().String()
	sinceDescription = "Since parameter to changes feed"
	since            = kingpin.Arg("since", sinceDescription).String()
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
	if *uqUrl == "" {
		kingpin.UsageErrorf("uqURL is empty")
		return
	}

	officeRadarApp := officeradar.NewOfficeRadarApp(*sgUrl, *uqUrl)
	err := officeRadarApp.InitApp()
	if err != nil {
		logg.LogPanic("Error initializing officeradar app: %v", err)
	}
	go officeRadarApp.FollowChangesFeed(*since)

	select {} // block forever

}
