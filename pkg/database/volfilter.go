package database

import "reflect"

type VolumeFilter struct {
	Page    int
	PerPage int

	NotDeleted bool `filter:"not_deleted"`
	Deleted    bool `filter:"deleted"`
}

var volFilterCache = make(map[string]int)

func init() {
	t := reflect.TypeOf(VolumeFilter{})
	for i := 0; i < t.NumField(); i++ {
		tag, ok := t.Field(i).Tag.Lookup("filter")
		if !ok {
			continue
		}
		volFilterCache[tag] = i
	}
}

func ParseVolumeFilter(filters ...string) VolumeFilter {
	var ret VolumeFilter
	v := reflect.ValueOf(&ret).Elem()
	for _, filter := range filters {
		if field, ok := volFilterCache[filter]; ok {
			v.Field(field).SetBool(true)
		}
	}
	return ret
}
