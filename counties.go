package counties

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"

	geo "github.com/kellydunn/golang-geo"
	"github.com/tidwall/rtree"
)

const (
	// CountyJSONFile is the default data source for county polygon info
	// as generated from github.com/paulstuart/counties
	CountyJSONFile = "county_poly.json"
	CountyGOBFile  = "county_geo.gob.gz"
)

// Point represents Lat,Lon
type Point [2]float64

// Points is a collection of geographic locations
type Points []Point

// Rect is the min and max verticies of a bounding box
type Rect [2]Point

// CountyGeo is ready for consumption
type CountyGeo struct {
	GeoID int    `json:"geoid"`
	Name  string `json:"name"`
	Full  string `json:"fullname"`
	State string `json:"state"`
	BBox  Rect   `json:"bbox"`
	Poly  Points `json:"polygon"`
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
	GeoID    int
	County   string
	Fullname string
	State    string
}

var (
	// polygons are indexed by GeoID
	countyLookupPolygons = make(map[int][]*geo.Polygon)

	// locations are indexed by GeoID
	CountyLookupMeta = make(map[int]Location)

	// countyLookupBBoxen contains all the bounding boxes for each US county
	// the bboxen are linked to a GeoID, and then the possible hits
	// are confirmed in countyLookupPolygons
	countyLookupBBoxen rtree.RTree
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

func toGeoPoly(pts Points) *geo.Polygon {
	points := make([]*geo.Point, len(pts))
	for i, pt := range pts {
		points[i] = geo.NewPoint(pt[1], pt[0])
	}

	return geo.NewPolygon(points)
}

func boundingBox(pp Points) [2]Point {
	var maxX, maxY, minX, minY float64

	for _, pt := range pp {
		if pt[0] > maxX || maxX == 0.0 {
			maxX = pt[0]
		}
		if pt[1] > maxY || maxY == 0.0 {
			maxY = pt[1]
		}
		if pt[0] < minX || minX == 0.0 {
			minX = pt[0]
		}
		if pt[1] < minY || minY == 0.0 {
			minY = pt[1]
		}
	}

	return [2]Point{
		{minX, minY},
		{maxX, maxY},
	}
}

// InitCountyLookup prepares data for searching for counties
func InitCountyLookup(counties []CountyGeo) {
	for _, county := range counties {
		// recalculate the bounding box, as the sourced version is not accurate enough
		bbox := boundingBox(county.Poly)
		countyLookupPolygons[county.GeoID] = append(countyLookupPolygons[county.GeoID], toGeoPoly(county.Poly))
		countyLookupBBoxen.Insert(bbox[0], bbox[1], county.GeoID)
		CountyLookupMeta[county.GeoID] = Location{Name: county.Name, FullName: county.Full, State: county.State}
	}
}

// LoadCachedCountyGeo uses GOB encoded geodata
// to build county lookup functions
func LoadCachedCountyGeo(filename string) error {
	var counties []CountyGeo
	err := GobLoad(filename, &counties)
	if err != nil {
		return err
	}
	InitCountyLookup(counties)
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

	load.BBox = boundingBox(load.Poly)
	return load, nil
}

// FindCounty returns the county associated with the given location
func FindCounty(lat, lon float64) (CountyMeta, error) {
	// NOTE: the polygon coordinates are in form of lon,lat
	// TODO: unify that to lat,lon
	pts := Point{lon, lat}
	foundGeo := -1
	var possible []int

	countyLookupBBoxen.Search(pts, pts,
		func(min, max [2]float64, value interface{}) bool {
			id := value.(int)
			polys, ok := countyLookupPolygons[id]
			if !ok {
				log.Println("no polygon for geoid:", id)
				return true
			}
			for _, poly := range polys {
				in := geo.NewPoint(lat, lon)
				if poly.Contains(in) {
					foundGeo = id
					return false
				}
				possible = append(possible, id)
			}
			return true
		})

	// TODO: add centroids data and do triagulation if
	//       no direct hit but multiple possible
	if foundGeo == -1 && len(possible) == 1 {
		foundGeo = possible[0]
	}
	location, ok := CountyLookupMeta[foundGeo]
	if !ok {
		return CountyMeta{}, fmt.Errorf("could not find county at (%5f,%5f)", lat, lon)
	}

	meta := CountyMeta{
		GeoID:    foundGeo,
		County:   location.Name,
		Fullname: location.FullName,
		State:    location.State,
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
