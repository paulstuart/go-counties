# go-counties
Geo lookup for US Counties

This library uses the data as generated from [https://github.com/paulstuart/counties](County Data) to identify the U.S. county by lat,lon.

Performance is aided by first identifying possible candidates by bounding box, and only checking the possible enclosing polygons of resulting candidates.