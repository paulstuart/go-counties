package counties

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/paulstuart/polygons"
	"golang.org/x/exp/constraints"
)

const (
	// CountyJSONFile is the default data source for county polygon info
	// as generated from github.com/paulstuart/counties
	CountyJSONFile = "county_poly.json"
	CountyGOBFile  = "county_geo.gob.gz"
	CountyPolyFile = "county_poly.gob.gz"
	CountyMetaFile = "county_meta.gob.gz"
)

// Point represents Lat,Lon
//type Point [2]float64
type Point = polygons.Pair

// Points is a collection of geographic locations
//type Points []Point
type Points = polygons.PPoints

// Rect is the min and max verticies of a bounding box
type Rect = polygons.BBox //[2]Point

// CountyGeo is ready for consumption
type CountyGeo struct {
	Name  string `json:"name"`
	Full  string `json:"fullname"`
	State string `json:"state"`
	Poly  Points `json:"polygon"`
	BBox  Rect   `json:"bbox"`
	GeoID int    `json:"geoid"`
}

func (cg CountyGeo) Meta() CountyMeta {
	return CountyMeta{
		GeoID:    cg.GeoID,
		County:   cg.Name,
		Fullname: cg.Full,
		State:    cg.State,
	}
}

// Location is name and state of the county
type Location struct {
	Name     string
	FullName string
	State    string
}

// countyRawJSON represents data as generated from github.com/paulstuart/counties
// TODO: let's add centroids
type countyRawJSON struct {
	GeoID   string `json:"geoid"`
	Full    string `json:"fullname"`
	Name    string `json:"name"`
	State   string `json:"state"`
	GeoType string `json:"geotype"`
	BBox    string `json:"bbox"`
	Poly    string `json:"poly"`
}

// CountyMeta is the principle county meta data
type CountyMeta struct {
	County   string
	Fullname string
	State    string
	GeoID    int
}

var (
	// polygons are indexed by GeoID
	//countyLookupPolygons = make(map[int][]*geo.Polygon)

	// locations are indexed by GeoID
	CountyLookupMeta = make(map[int]Location)

	// countyLookupBBoxen contains all the bounding boxes for each US county
	// the bboxen are linked to a GeoID, and then the possible hits
	// are confirmed in countyLookupPolygons
	finder polygons.Finder[uint]
)

/*
	County lookup strategy

	To find which county a point exists in, first <countyLookupBBoxen> is queried for all counties
	where the bounding box(es) of the area contain that point.

	For example in the west side of the SFBay, a particular point could lie in either
	San Francisco, Alameda, or Contra Costa county (IIRC)

	Once the candidate counties are identified, their respective polygons are examined to see
	which one actually contains that point.
*/

/*
// TODO: alias ppoints
func (pp Points) convert() polygons.PPoints {
	pgp := make(polygons.PPoints, len(pp))
	for i, pt := range pp {
		pgp[i] = polygons.Pair(pt)
	}
	return pgp
}
*/

// InitCountyLookup prepares data for searching for counties
func InitCountyLookup(counties []CountyGeo) {
	finder = *polygons.NewFinder[uint]()
	for _, county := range counties {
		switch county.State {
		case "AS", "GU", "VI", "MP":
			continue // skip American Somoa, Guam, Virgin Islands, Northern Mariana Islands
		}
		finder.Add(county.GeoID, county.Poly)
		CountyLookupMeta[county.GeoID] = Location{Name: county.Name, FullName: county.Full, State: county.State}
	}
}

func SaveMeta(filename string) error {
	return GobDump(filename, CountyLookupMeta)
}

func LoadMeta(filename string) error {
	return GobLoad(filename, &CountyLookupMeta)
}

func NewSearcher[T constraints.Unsigned](f polygons.Finder[T]) polygons.Searcher[T] {
	return polygons.NewSearcher(&f)
}

func SaveSearcher(filename string) error {
	_initLookups()
	s := NewSearcher(finder)
	return GobDump(filename, s)
}

// LoadCachedCountyGeo uses GOB encoded geodata
// to build county lookup functions
func LoadCachedCountyGeo(filename string) error {
	now := time.Now()
	var counties []CountyGeo
	err := GobLoad(filename, &counties)
	if err != nil {
		return err
	}
	InitCountyLookup(counties)
	log.Printf("elapsed: %s", time.Since(now))
	return nil
}

// LoadCountyJSON loads the json dump file from the data as prepared using
// https://github.com/paulstuart/counties
func LoadCountyJSON(filename string) ([]CountyGeo, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	var raw []countyRawJSON
	dec := json.NewDecoder(f)
	if err := dec.Decode(&raw); err != nil {
		return nil, err
	}
	loaded := make([]CountyGeo, len(raw))
	for i, src := range raw {
		load, err := src.Load()
		if err != nil {
			return nil, fmt.Errorf("failed for (%d/%d): %w", i+1, len(raw), err)
		}
		loaded[i] = load
	}
	return loaded, nil
}

// Load converts the serialize geodata into usable data
func (c countyRawJSON) Load() (CountyGeo, error) {
	geoid, err := strconv.Atoi(c.GeoID)
	if err != nil {
		return CountyGeo{}, err
	}
	load := CountyGeo{
		GeoID: geoid,
		Name:  c.Name,
		Full:  c.Full,
		State: c.State,
	}

	var polybox Points
	if err := json.Unmarshal([]byte(c.BBox), &polybox); err != nil {
		return load, fmt.Errorf("bbox geoid: %s -- %w", c.GeoID, err)
	}
	if len(polybox) < 5 {
		return load, fmt.Errorf("geoid %q has incomplete bbox (%d)", c.GeoID, len(polybox))
	}

	if err := json.Unmarshal([]byte(c.Poly), &load.Poly); err != nil {
		return load, fmt.Errorf("poly geoid: %s -- %w", c.GeoID, err)
	}

	load.BBox = load.Poly.BBox()
	return load, nil
}

var ErrNotFound = errors.New("not found")

func CountyCount() int {
	return finder.Size()
}

// FindCounty returns the county associated with the given location
func FindCounty(lat, lon float64) (CountyMeta, error) {
	// NOTE: the polygon coordinates are in form of lon,lat
	// TODO: unify that to lat,lon
	pts := Point{lon, lat}
	idx, dist := finder.Search(pts)
	if idx < 0 {
		return CountyMeta{}, ErrNotFound
	}
	location := CountyLookupMeta[idx]
	meta := CountyMeta{
		GeoID:    idx,
		State:    location.State,
		County:   location.Name,
		Fullname: location.FullName,
	}
	if dist > 0 {
		log.Printf("closest county to %.6f,%6f (%f) is %s, %s", lat, lon, dist, location.Name, location.State)
	}

	return meta, nil
}

// SearchCounty returns the county associated with the given location
func SearchCounty(lat, lon float64, s *polygons.Searcher[uint]) (CountyMeta, error) {
	// NOTE: the polygon coordinates are in form of lon,lat
	// TODO: unify that to lat,lon
	pts := Point{lon, lat}
	fmt.Printf("SEARCH: %v\n", pts)
	idx, dist := s.Search(pts)
	if idx < 0 {
		fmt.Printf("IDX: %d", idx)
		return CountyMeta{}, ErrNotFound
	}
	location := CountyLookupMeta[idx]
	meta := CountyMeta{
		GeoID:    idx,
		State:    location.State,
		County:   location.Name,
		Fullname: location.FullName,
	}
	if dist > 0 {
		log.Printf("closest county to %.6f,%6f (%f) is %s, %s", lat, lon, dist, location.Name, location.State)
	}

	return meta, nil
}

// ProcessJSONData loads the json data file from the data as sourced from https://github.com/paulstuart/counties
// It will save it as a GOB datafile for faster loading
func ProcessJSONData(source, saved string) error {
	loaded, err := LoadCountyJSON(source)
	if err != nil {
		return fmt.Errorf("failed to process %q -- %w", source, err)
	}
	return GobDump(saved, loaded)
}
