package counties

import (
	"fmt"
	"sync"

	"github.com/paulstuart/rtree"
)

type Lookup struct {
}

func foo() {

}

var (
	_initMu sync.Mutex
	_initOk bool
)

func _initLookups() {
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

func ReadOnlyLookup(filename string) (*Lookup, error) {
	_initLookups()
	var rt rtree.Generic[int]
	fmt.Printf("RTREE: %+v\n", rt)
	return nil, nil
}
