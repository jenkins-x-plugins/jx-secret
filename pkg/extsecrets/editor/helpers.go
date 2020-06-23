package editor

import "sort"

// SortPropertyValues sorts the property values in property name order
func SortPropertyValues(pvs []PropertyValue) {
	sort.Slice(pvs, func(i, j int) bool {
		pv1 := pvs[i]
		pv2 := pvs[j]
		return pv1.Property < pv2.Property
	})
}
