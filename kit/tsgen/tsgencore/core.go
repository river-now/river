package tsgencore

import (
	"fmt"
	"reflect"
	"regexp"
	"slices"
)

type IDStr = string
type _results = map[IDStr]*TypeInfo

type TypeInfo struct {
	OriginalName string
	ResolvedName string
	ReflectType  reflect.Type
	TSStr        string

	_id IDStr
}

var _any any
var _null_id = getID(&AdHocType{TypeInstance: nil})
var _unknown_id = getID(&AdHocType{TypeInstance: &_any}) // ptr intentional to get interface {}
func (t *TypeInfo) IsTSNull() bool                       { return t._id == _null_id }
func (t *TypeInfo) IsTSUnknown() bool                    { return t._id == _unknown_id }
func (t *TypeInfo) IsTSBasicType() bool                  { return isBasicType(t.ReflectType) }

func getEffectiveReflectType(instance any) reflect.Type {
	t := reflect.TypeOf(instance)
	if t != nil && t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

func getEffectiveRequestedName(t reflect.Type, requestedName string) string {
	if requestedName != "" {
		return requestedName
	}
	return getNaturalName(t)
}

func getNaturalName(t reflect.Type) string {
	if t != nil {
		n := t.Name()
		if n != "" && isBasicType(t) {
			return ""
		}
		return n
	}
	return ""
}

func traverseType(adHocType *AdHocType) (_results, IDStr) {
	if adHocType == nil || adHocType.TypeInstance == nil {
		return _results{}, ""
	}

	t := getEffectiveReflectType(adHocType.TypeInstance)

	effectiveRequestedName := getEffectiveRequestedName(t, adHocType.TSTypeName)

	c := newTypeCollector()
	c.rootType = t
	c.rootRequestedName = effectiveRequestedName

	c.collectType(t, adHocType.TSTypeName)
	return c.buildDefinitions()
}

type Results struct {
	Types     []*TypeInfo
	id_to_idx map[IDStr]int
}

func (m *Results) GetTypeInfo(adHocType *AdHocType) *TypeInfo {
	// If we have it, return it
	id := getID(adHocType)
	if idx, ok := m.id_to_idx[id]; ok {
		return m.Types[idx]
	}

	// If we don't, you're probably looking for a basic type,
	// which will fall back to "unknown" if not
	reflectType := getEffectiveReflectType(adHocType.TypeInstance)
	return &TypeInfo{
		_id:          id,
		OriginalName: adHocType.TSTypeName,
		ResolvedName: "",
		ReflectType:  reflectType,
		TSStr:        getBasicTSType(reflectType),
	}
}

func mergeTypeResults(results ..._results) Results {
	if len(results) == 0 {
		return Results{}
	}

	flattened := make(map[IDStr]*TypeInfo, len(results))
	nameUsageCounter := make(map[string]int, len(results))

	for _, result := range results {
		for id, typeInfo := range result {
			flattened[id] = typeInfo
			if typeInfo.OriginalName != "" {
				nameUsageCounter[typeInfo.OriginalName]++
			}
		}
	}

	// Create a slice of IDs for deterministic ordering of suffixed var names
	ids := make([]IDStr, 0, len(flattened))
	for id := range flattened {
		ids = append(ids, id)
	}

	slices.SortFunc(ids, func(i, j string) int {
		if i < j {
			return -1
		}
		if i > j {
			return 1
		}
		return 0
	})

	finalTypes := make([]*TypeInfo, 0, len(flattened))
	id_to_idx := make(map[string]int, len(flattened))
	nameVersions := make(map[string]int, len(flattened))

	for i, id := range ids {
		typeInfo := flattened[id]
		effectiveName := typeInfo.OriginalName

		if effectiveName != "" && nameUsageCounter[effectiveName] > 1 {
			nameVersions[effectiveName]++
			version := nameVersions[effectiveName]

			if version > 1 {
				effectiveName = fmt.Sprintf("%s_%d", effectiveName, version)
			}
		}

		typeInfo.ResolvedName = effectiveName

		id_to_idx[typeInfo._id] = i
		finalTypes = append(finalTypes, typeInfo)
	}

	for i, typeInfo := range finalTypes {
		finalTypes[i].TSStr = idRegex.ReplaceAllStringFunc(typeInfo.TSStr, func(id string) string {
			if idx, ok := id_to_idx[id]; ok {
				return finalTypes[idx].ResolvedName
			}
			panic("tsgencore error: could not find resolved name to replace matched id: " + id)
		})
	}

	return Results{
		Types:     finalTypes,
		id_to_idx: id_to_idx,
	}
}

var idRegex = regexp.MustCompile(`\$tsgen\$[^$]+\$tsgen\$`)

type AdHocType = struct {
	// Instance of the struct to generate TypeScript for
	TypeInstance any
	// Name is required only if type is anonymous, otherwise optional override
	TSTypeName string
}

func ProcessTypes(adHocTypes []*AdHocType) Results {
	types := make([]_results, 0, len(adHocTypes))

	for _, adHocType := range adHocTypes {
		result, _ := traverseType(adHocType)
		types = append(types, result)
	}

	return mergeTypeResults(types...)
}

func getID(adHocType *AdHocType) IDStr {
	t := getEffectiveReflectType(adHocType.TypeInstance)
	return getIDFromReflectType(t, adHocType.TSTypeName)
}

func getIDFromReflectType(t reflect.Type, requestedName string) IDStr {
	natural_name := getNaturalName(t)
	effective_requested_name := getEffectiveRequestedName(t, requestedName)
	if effective_requested_name != "" && effective_requested_name != natural_name {
		return fmt.Sprintf("$tsgen$%v+%s$tsgen$", t, requestedName)
	}
	return fmt.Sprintf("$tsgen$%v$tsgen$", t)
}
