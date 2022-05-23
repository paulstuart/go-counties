module github.com/paulstuart/go-counties

go 1.18

require (
	github.com/paulstuart/polygons v0.0.0-20220511061739-aa0cd61a5d64
	github.com/paulstuart/rtree v1.4.2-0.20220430215825-ea1b5d015948
	github.com/stretchr/testify v1.7.1
	golang.org/x/exp v0.0.0-20220428152302-39d4317da171
)

require (
	github.com/davecgh/go-spew v1.1.0 // indirect
	github.com/edsrzf/mmap-go v1.1.0 // indirect
	github.com/paulstuart/geo v0.0.0-20220410181904-83d5586f49f5 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/tidwall/geoindex v1.6.1 // indirect
	github.com/tidwall/mmap v0.2.1 // indirect
	github.com/tidwall/rtree v1.5.0 // indirect
	golang.org/x/sys v0.0.0-20220513210249-45d2b4557a2a // indirect
	gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c // indirect
)

//replace github.com/paulstuart/rtree => ../rtree // github.com/paulstuart/polygons -> github.com/tidwall/polygons

//replace github.com/tidwall/rtree => ../rtree // github.com/paulstuart/polygons -> github.com/tidwall/polygons
//replace github.com/paulstuart/polygons => ../polygons
