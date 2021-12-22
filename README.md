# go-counties
Geo lookup for US Counties

This library uses the output generated from [US County Data](https://github.com/paulstuart/counties) to identify the U.S. county by lat,lon.

Performance is aided by first identifying possible candidates by bounding box, and only checking the possible enclosing polygons of resulting candidates.
