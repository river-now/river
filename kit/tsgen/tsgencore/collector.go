package tsgencore

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

type TSTyper interface {
	TSType() map[string]string
}

type typeCollector struct {
	types             map[reflect.Type]*typeEntry
	rootType          reflect.Type
	rootRequestedName string
}

type typeEntry struct {
	originalGoType reflect.Type
	resolvedName   string
	usedAsEmbedded bool
	isReferenced   bool
	visited        bool
	coreType       string
	requestedName  string
}

func newTypeCollector() *typeCollector {
	return &typeCollector{types: make(map[reflect.Type]*typeEntry)}
}

func (c *typeCollector) getOrCreateEntry(t reflect.Type, userDefinedAlias ...string) *typeEntry {
	if entry, exists := c.types[t]; exists {
		if t == c.rootType && c.rootRequestedName != "" && entry.requestedName == "" {
			entry.requestedName = c.rootRequestedName
		}
		return entry
	}

	entry := &typeEntry{originalGoType: t}
	if t == c.rootType && c.rootRequestedName != "" {
		entry.requestedName = c.rootRequestedName
	} else if len(userDefinedAlias) > 0 && userDefinedAlias[0] != "" {
		entry.requestedName = userDefinedAlias[0]
	} else {
		var requestedName string
		if !isBasicType(t) {
			requestedName = toSanitizedName(t)
		}
		if requestedName != "" {
			entry.requestedName = requestedName
		}
	}

	c.types[t] = entry
	return entry
}

func (c *typeCollector) collectType(t reflect.Type, userDefinedAlias ...string) {
	isRoot := (t == c.rootType)

	if t.Name() != "" || isRoot {
		entry := c.getOrCreateEntry(t, userDefinedAlias...)
		if entry.visited {
			return
		}
		entry.visited = true

		if !isRoot && isBasicType(t) {
			entry.coreType = c.getTypeScriptType(t)
			return
		}
	} else {
		if !isRoot && isBasicType(t) {
			return
		}
	}

	switch t.Kind() {
	case reflect.Struct:
		c.collectStructFields(t)

	case reflect.Ptr:
		if t.Name() != "" {
			c.getOrCreateEntry(t, userDefinedAlias...)
		}
		c.collectType(t.Elem())

	case reflect.Slice, reflect.Array:
		if t.Name() != "" {
			c.getOrCreateEntry(t, userDefinedAlias...)
		}
		c.collectType(t.Elem())

	case reflect.Map:
		if t.Name() != "" {
			c.getOrCreateEntry(t, userDefinedAlias...)
		}
		c.collectType(t.Key())
		c.collectType(t.Elem())
	}
}

func (c *typeCollector) collectStructFields(t reflect.Type) {
	for i := range t.NumField() {
		field := t.Field(i)
		if isUnexported(field) {
			continue
		}
		fieldType := field.Type

		if field.Anonymous {
			jsonTag := field.Tag.Get("json")
			hasJsonTag := jsonTag != "" && jsonTag != "-"
			if fieldType.Kind() == reflect.Struct {
				embeddedEntry := c.getOrCreateEntry(fieldType)
				embeddedEntry.usedAsEmbedded = true
				if hasJsonTag {
					embeddedEntry.isReferenced = true
				}
				c.collectType(fieldType)
				continue
			}
			if fieldType.Kind() == reflect.Ptr && fieldType.Elem().Kind() == reflect.Struct {
				embeddedType := fieldType.Elem()
				embeddedEntry := c.getOrCreateEntry(embeddedType)
				embeddedEntry.usedAsEmbedded = true
				if hasJsonTag {
					embeddedEntry.isReferenced = true
				}
				c.collectType(embeddedType)
				continue
			}
		}

		c.collectFieldType(fieldType)
	}
}

func (c *typeCollector) collectFieldType(t reflect.Type) {
	switch t.Kind() {
	case reflect.Struct:
		entry := c.getOrCreateEntry(t)
		entry.isReferenced = true
		c.collectType(t)

	case reflect.Ptr:
		if t.Name() != "" {
			entry := c.getOrCreateEntry(t)
			entry.isReferenced = true
		}
		if t.Elem().Kind() == reflect.Struct {
			entry := c.getOrCreateEntry(t.Elem())
			entry.isReferenced = true
		}
		c.collectType(t.Elem())

	case reflect.Slice, reflect.Array:
		if t.Name() != "" {
			entry := c.getOrCreateEntry(t)
			entry.isReferenced = true
		}
		elemType := t.Elem()
		if elemType.Kind() == reflect.Struct {
			entry := c.getOrCreateEntry(elemType)
			entry.isReferenced = true
		} else if elemType.Kind() == reflect.Ptr && elemType.Elem().Kind() == reflect.Struct {
			entry := c.getOrCreateEntry(elemType.Elem())
			entry.isReferenced = true
		}
		c.collectType(elemType)

	case reflect.Map:
		if t.Name() != "" {
			entry := c.getOrCreateEntry(t)
			entry.isReferenced = true
		}

		// Map key
		keyType := t.Key()
		if keyType.Kind() == reflect.Struct {
			entry := c.getOrCreateEntry(keyType)
			entry.isReferenced = true
		}
		c.collectType(keyType)

		// Map value
		valueType := t.Elem()
		if valueType.Kind() == reflect.Struct {
			entry := c.getOrCreateEntry(valueType)
			entry.isReferenced = true
		} else if valueType.Kind() == reflect.Ptr && valueType.Elem().Kind() == reflect.Struct {
			entry := c.getOrCreateEntry(valueType.Elem())
			entry.isReferenced = true
		}
		c.collectType(valueType)
	}
}

func (c *typeCollector) buildDefinitions() (_results, IDStr) {
	if len(c.types) > 0 && c.rootType != nil {
		hasStructs := false
		for t := range c.types {
			if t == nil {
				continue
			}
			if t.Kind() == reflect.Struct {
				hasStructs = true
				break
			}
		}

		if !hasStructs {
			id := getIDFromReflectType(c.rootType, c.rootRequestedName)

			results := map[IDStr]*TypeInfo{id: {
				_id:          id,
				OriginalName: c.rootRequestedName,
				ResolvedName: c.types[c.rootType].resolvedName,
				ReflectType:  c.rootType,
				TSStr:        c.getTypeScriptType(c.rootType),
			}}

			return results, id
		}
	}

	for t, entry := range c.types {
		if entry.coreType == "" {
			if t.Kind() == reflect.Struct {
				fields := c.generateStructTypeFields(t)
				entry.coreType = buildObj(fields)
			} else {
				entry.coreType = c.getTypeScriptType(t)
			}
		}
	}

	reflectTypeToID := make(map[reflect.Type]IDStr)

	for t, entry := range c.types {
		if t.Kind() == reflect.Struct {
			if t != c.rootType && entry.usedAsEmbedded && !entry.isReferenced {
				continue
			}
		}

		requestedName := entry.requestedName
		if t == c.rootType && c.rootRequestedName != "" {
			requestedName = c.rootRequestedName
		}

		id := getIDFromReflectType(t, requestedName)
		reflectTypeToID[t] = id
	}

	finalTypes := make(map[IDStr]*TypeInfo)

	for t, id := range reflectTypeToID {
		requestedName := ""

		entry := c.types[t]

		if t == c.rootType {
			requestedName = c.rootRequestedName
		} else if entry.requestedName != "" {
			requestedName = entry.requestedName
		}

		finalTypes[id] = &TypeInfo{
			_id:          id,
			OriginalName: requestedName,
			ResolvedName: entry.resolvedName,
			ReflectType:  t,
			TSStr:        c.types[t].coreType,
		}
	}

	if c.rootType != nil {
		return finalTypes, reflectTypeToID[c.rootType]
	}

	panic("tsgencore error: something went wrong")
}

func (c *typeCollector) generateStructTypeFields(t reflect.Type) []string {
	if t.Kind() != reflect.Struct {
		return nil
	}

	tsTypeMap := getTSTypeMap(t)
	var fields []string

	for i := range t.NumField() {
		field := t.Field(i)
		if field.PkgPath != "" {
			continue
		}
		if shouldOmitField(field) {
			continue
		}
		if field.Anonymous {
			if field.Type.Kind() == reflect.Struct {
				jsonTag := field.Tag.Get("json")
				if jsonTag != "" && jsonTag != "-" {
					structType := field.Type
					c.collectType(structType)
					embeddedEntry := c.getOrCreateEntry(structType)
					embeddedEntry.isReferenced = true
					jsonFieldName := strings.Split(jsonTag, ",")[0]
					fields = append(fields, fmt.Sprintf("%s: %s", jsonFieldName, structType.Name()))
					continue
				}
				localTSTypeMap := getTSTypeMap(field.Type)
				for j := range field.Type.NumField() {
					embField := field.Type.Field(j)
					if embField.PkgPath != "" || shouldOmitField(embField) {
						continue
					}
					jsonFieldName := getJSONFieldName(embField)
					if jsonFieldName == "" {
						continue
					}
					var customType string
					if localTSTypeMap[embField.Name] != "" { // lookup by original name
						customType = localTSTypeMap[embField.Name]
					} else {
						customType = get_ts_type_from_struct_tag(embField)
					}
					var fieldType string
					if customType != "" {
						fieldType = customType
					} else {
						fieldType = c.getTypeScriptType(embField.Type)
					}
					if isOptionalField(embField) {
						fields = append(fields, fmt.Sprintf("%s?: %s", jsonFieldName, fieldType))
					} else {
						fields = append(fields, fmt.Sprintf("%s: %s", jsonFieldName, fieldType))
					}
				}
				continue
			} else if field.Type.Kind() == reflect.Ptr && field.Type.Elem().Kind() == reflect.Struct {
				jsonTag := field.Tag.Get("json")
				if jsonTag != "" && jsonTag != "-" {
					ptrType := field.Type.Elem()
					c.collectType(ptrType)
					embeddedEntry := c.getOrCreateEntry(ptrType)
					embeddedEntry.isReferenced = true
					jsonFieldName := strings.Split(jsonTag, ",")[0]
					fields = append(fields, fmt.Sprintf("%s?: %s", jsonFieldName, ptrType.Name()))
					continue
				}
				ptrType := field.Type.Elem()
				localTSTypeMap := getTSTypeMap(ptrType)
				for j := range ptrType.NumField() {
					embField := ptrType.Field(j)
					if embField.PkgPath != "" || shouldOmitField(embField) {
						continue
					}
					jsonFieldName := getJSONFieldName(embField)
					if jsonFieldName == "" {
						continue
					}
					var customType string
					if localTSTypeMap[embField.Name] != "" {
						customType = localTSTypeMap[embField.Name]
					} else {
						customType = get_ts_type_from_struct_tag(embField)
					}
					var fieldType string
					if customType != "" {
						fieldType = customType
					} else {
						fieldType = c.getTypeScriptType(embField.Type)
					}
					fields = append(fields, fmt.Sprintf("%s?: %s", jsonFieldName, fieldType))
				}
				continue
			}
		}

		jsonFieldName := getJSONFieldName(field)
		if jsonFieldName == "" {
			continue
		}

		var customType string
		if tsTypeMap[field.Name] != "" { // lookup by original name
			customType = tsTypeMap[field.Name]
		} else {
			customType = get_ts_type_from_struct_tag(field)
		}

		var fieldType string
		if customType != "" {
			fieldType = customType
		} else {
			if field.Type.Kind() == reflect.Ptr {
				fieldType = c.getTypeScriptType(field.Type.Elem())
			} else {
				fieldType = c.getTypeScriptType(field.Type)
			}
		}

		if isOptionalField(field) {
			fields = append(fields, fmt.Sprintf("%s?: %s", jsonFieldName, fieldType))
		} else {
			fields = append(fields, fmt.Sprintf("%s: %s", jsonFieldName, fieldType))
		}
	}

	return fields
}

func getBasicTSType(t reflect.Type) string {
	if t == nil {
		return "null"
	}

	switch t.Kind() {
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return "number"
	case reflect.String:
		return "string"
	default:
		return "unknown"
	}
}

func (c *typeCollector) getTypeScriptType(t reflect.Type) string {
	if t == nil {
		return "null"
	}

	var typeStr string

	switch t.Kind() {
	case reflect.Interface:
		typeStr = "unknown"

	case reflect.Bool:
		typeStr = "boolean"

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		typeStr = "number"

	case reflect.String:
		typeStr = "string"

	case reflect.Ptr:
		typeStr = c.getTypeScriptType(t.Elem())

	case reflect.Slice, reflect.Array:
		elemType := c.getTypeScriptType(t.Elem())
		typeStr = fmt.Sprintf("Array<%s>", elemType)

	case reflect.Map:
		keyType := c.getTypeScriptType(t.Key())
		valueType := c.getTypeScriptType(t.Elem())
		typeStr = fmt.Sprintf("Record<%s, %s>", keyType, valueType)

	case reflect.Struct:
		switch {
		case t == reflect.TypeOf(time.Time{}):
			typeStr = "string"
		case t == reflect.TypeOf(time.Duration(0)):
			typeStr = "number"
		case t.Name() != "" && c.types[t] != nil:
			entry := c.getOrCreateEntry(t)
			requestedName := entry.requestedName

			if t == c.rootType && c.rootRequestedName != "" {
				requestedName = c.rootRequestedName
			}

			// ID will be replaced later with the correct resolved name
			typeStr = getIDFromReflectType(t, requestedName)
		default:
			fields := c.generateStructTypeFields(t)
			typeStr = buildObj(fields)
		}

	default:
		typeStr = "unknown"
	}

	return typeStr
}

func isBasicType(t reflect.Type) bool {
	if t == nil {
		return false
	}

	if t == reflect.TypeOf(time.Time{}) || t == reflect.TypeOf(time.Duration(0)) {
		return true
	}

	switch t.Kind() {
	case reflect.Interface,
		reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64,
		reflect.String:
		return true
	default:
		return false
	}
}

func isUnexported(field reflect.StructField) bool {
	return field.PkgPath != ""
}

func isOptionalField(field reflect.StructField) bool {
	if field.Type.Kind() == reflect.Ptr {
		return true
	}
	tag := field.Tag.Get("json")
	if tag != "" {
		parts := strings.Split(tag, ",")
		for _, part := range parts[1:] {
			if part == "omitempty" || part == "omitzero" {
				return true
			}
		}
	}
	return false
}

func getJSONFieldName(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag == "" {
		return field.Name
	}
	parts := strings.Split(tag, ",")
	if parts[0] == "-" {
		return ""
	}
	if parts[0] != "" {
		return parts[0]
	}
	return field.Name
}

func shouldOmitField(field reflect.StructField) bool {
	tag := field.Tag.Get("json")
	return tag == "-" || strings.HasPrefix(tag, "-,")
}

func get_ts_type_from_struct_tag(field reflect.StructField) string {
	return field.Tag.Get("ts_type")
}

func buildObj(fields []string) string {
	if len(fields) == 0 {
		return "Record<never, never>"
	}
	var sb strings.Builder
	sb.WriteString("{\n")
	for _, field := range fields {
		sb.WriteString("\t")
		sb.WriteString(field)
		sb.WriteString(";\n")
	}
	sb.WriteString("}")
	return sb.String()
}

func getTSTypeMap(t reflect.Type) map[string]string {
	if !implementsTSTyper(t) {
		return nil
	}
	var instance reflect.Value
	if t.Kind() == reflect.Ptr {
		instance = reflect.New(t.Elem())
	} else {
		instance = reflect.New(t)
	}
	initializeEmbeddedPointers(instance)
	if t.Kind() == reflect.Ptr {
		return instance.Interface().(TSTyper).TSType()
	} else {
		return instance.Interface().(TSTyper).TSType()
	}
}

func initializeEmbeddedPointers(v reflect.Value) {
	if v.Kind() == reflect.Ptr && v.Elem().Kind() == reflect.Struct {
		elem := v.Elem()
		typ := elem.Type()
		for i := range elem.NumField() {
			field := elem.Field(i)
			fieldType := typ.Field(i)
			if fieldType.Anonymous && field.Kind() == reflect.Ptr && field.IsNil() {
				newValue := reflect.New(field.Type().Elem())
				field.Set(newValue)
				initializeEmbeddedPointers(newValue)
			}
		}
	}
}

func implementsTSTyper(t reflect.Type) bool {
	if t.Implements(reflect.TypeOf((*TSTyper)(nil)).Elem()) {
		return true
	}
	if t.Kind() != reflect.Ptr && reflect.PointerTo(t).Implements(reflect.TypeOf((*TSTyper)(nil)).Elem()) {
		return true
	}
	return false
}
