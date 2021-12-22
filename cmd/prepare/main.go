package main

import (
	"flag"
	"log"

	counties "github.com/paulstuart/go-counties"
)

var (
	jsonFile = counties.CountyJSONFile
	gobFile  = counties.CountyGOBFile
)

func main() {
	flag.StringVar(&jsonFile, "json", jsonFile, "json data comprising county lookup data")
	flag.StringVar(&gobFile, "gob", gobFile, "json data comprising county lookup data")
	flag.Parse()

	err := counties.ProcessJSONData(jsonFile, gobFile)
	if err != nil {
		log.Fatalf("can't open %q: %v", jsonFile, err)
	}
}
