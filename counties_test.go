package counties

import (
	"errors"
	"io/fs"
	"os"
	"testing"
	"time"
)

const (
	// Central & Grand, Alameda CA
	AlaLat   = 37.7703654
	AlaLon   = -122.2591581
	AlaGeoID = 6001
)

func exists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || !errors.Is(err, fs.ErrNotExist)
}

func TestProcessJSON(t *testing.T) {
	if exists(CountyGOBFile) {
		t.Skip("datafile already exists: " + CountyGOBFile)
	}
	err := ProcessJSONData(CountyJSONFile, CountyGOBFile)
	if err != nil {
		t.Fatal(err)
	}
}

func TestInitSummaries(t *testing.T) {
	err := LoadCachedCountyGeo(CountyGOBFile)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	meta, err := FindCounty(AlaLat, AlaLon)
	elapsed := time.Since(now)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("elapsed:%s meta:%+v", elapsed, meta)
}
