package counties

import (
	"encoding/csv"
	"errors"
	"io/fs"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/paulstuart/polygons"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// Central & Grand, Alameda CA
	AlaLat, AlaLon = 37.77033688841509, -122.25697282612731
	AlaGeoID       = 6001
)

var (
	initMu sync.Mutex
	initOk bool
)

func initLookups() {
	_initMu.Lock()
	if !_initOk {
		err := LoadCachedCountyGeo(CountyGOBFile)
		if err != nil {
			panic(err)
		}
		_initOk = true
	}
	_initMu.Unlock()
}

func exists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || !errors.Is(err, fs.ErrNotExist)
}

func TestProcessJSON(t *testing.T) {
	now := time.Now()
	// we cheat and use this to recreate the GOB file as needed
	if exists(CountyGOBFile) {
		t.Skip("datafile already exists: " + CountyGOBFile)
	}
	err := ProcessJSONData(CountyJSONFile, CountyGOBFile)
	require.NoError(t, err)
	t.Logf("processing time: %s", time.Since(now))
}

type lookupValid struct {
	lat, lon float64
	county   string
}

// 45.481300,-122.743996
// GEOID: 41051 POLYS: 1 MIN:[-122.929 45.4327] MAX:[-121.82 45.7287]
// Washington: [[-123.486 45.3172] [-122.744 45.7802]]

func TestLookup(t *testing.T) {
	initLookups()
	lookups := []lookupValid{
		{45.481300, -122.743996, "Washington"},
		{AlaLat, AlaLon, "Alameda"},
	}
	for _, look := range lookups {
		now := time.Now()
		meta, err := FindCounty(look.lat, look.lon)
		since := time.Since(now)
		require.NoError(t, err)
		assert.Equal(t, look.county, meta.County)
		t.Logf("LOOKUP Meta: %v, Elapsed: %s", meta, since)
	}
}

func TestSaveSearcher(t *testing.T) {
	initLookups()
	err := SaveSearcher(CountyPolyFile)
	require.NoError(t, err)
	err = SaveMeta(CountyMetaFile)
	require.NoError(t, err)
}

func TestLoadSearcher(t *testing.T) {
	//TestSaveSearcher(t)
	now := time.Now()
	var searcher polygons.Searcher[uint]
	err := GobLoad(CountyPolyFile, &searcher)
	t.Logf("load %d trees, elapsed: %s", searcher.Tree.Len(), time.Since(now))
	require.NoError(t, err)
	err = LoadMeta(CountyMetaFile)
	t.Logf("load total elapsed: %s", time.Since(now))
	require.NoError(t, err)
	lookups := []lookupValid{
		{45.481300, -122.743996, "Washington"},
		{AlaLat, AlaLon, "Alameda"},
	}
	for _, look := range lookups {
		now := time.Now()
		//meta, err := SearchCounty(look.lat, look.lon, &searcher)
		meta, err := SearchCounty(look.lon, look.lat, &searcher)
		since := time.Since(now)
		require.NoError(t, err)
		assert.Equal(t, look.county, meta.County)
		t.Logf("LOOKUP Meta: %v, Elapsed: %s", meta, since)
	}
	for _, look := range lookups {
		now := time.Now()
		meta, err := SearchCounty(look.lat, look.lon, &searcher)
		since := time.Since(now)
		require.NoError(t, err)
		assert.Equal(t, look.county, meta.County)
		t.Logf("LOOKUP Meta: %v, Elapsed: %s", meta, since)
	}
}

var (
	testLookups = []lookupValid{
		{45.481300, -122.743996, "Washington"},
		{AlaLat, AlaLon, "Alameda"},
	}
)

func TestUseSearcher(t *testing.T) {
	initLookups()
	searcher := NewSearcher(finder)
	//	searcher.Dump()
	for _, look := range testLookups {
		now := time.Now()
		meta, err := SearchCounty(look.lat, look.lon, &searcher)
		since := time.Since(now)
		require.NoError(t, err)
		assert.Equal(t, look.county, meta.County)
		t.Logf("LOOKUP Meta: %v, Elapsed: %s", meta, since)
	}
}

func TestEchoSearcher(t *testing.T) {
	lookups := []lookupValid{
		{45.481300, -122.743996, "Washington"},
		{AlaLat, AlaLon, "Alameda"},
	}
	initLookups()
	searcher := NewSearcher(finder)
	dupe := polygons.Echo(searcher)
	//	searcher.Dump()
	for _, look := range lookups {
		now := time.Now()
		meta, err := SearchCounty(look.lat, look.lon, &dupe)
		since := time.Since(now)
		require.NoError(t, err)
		assert.Equal(t, look.county, meta.County)
		t.Logf("LOOKUP Meta: %v, Elapsed: %s", meta, since)
	}
}

func parsePair(t *testing.T, ss []string) Point {
	t.Helper()
	lat, err := strconv.ParseFloat(ss[0], 64)
	require.NoError(t, err)
	lon, err := strconv.ParseFloat(ss[1], 64)
	require.NoError(t, err)
	return Point{lat, lon}
}

func TestProdMissing(t *testing.T) {
	initLookups()
	const sample = "testdata/county_not_found.csv"
	f, err := os.Open(sample)
	require.NoError(t, err)
	defer f.Close()
	c := csv.NewReader(f)
	recs, err := c.ReadAll()
	require.NoError(t, err)
	for _, rec := range recs {
		pt := parsePair(t, rec)
		// t.Logf("check pair: %v", pt)
		meta, err := FindCounty(pt[0], pt[1])
		if false {
			assert.NoError(t, err)
		}
		if err != nil {
			t.Logf("MISSING: %v -- META: %v ERR: %v", pt, meta, err)
		}
	}
}
