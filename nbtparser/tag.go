package nbtparser

import (
	"encoding/binary"
	"fmt"
	"math"
)

type TagType int

// documentation from http://web.archive.org/web/20110723210920/http://www.minecraft.net/docs/NBT.txt
const (
	TAG_End TagType = iota
	// Note: This tag is used to mark the end of a list.
	// Cannot be named! If type 0 appears where a Named Tag is expected, the name is assumed to be "".
	// (In other words, this Tag is always just a single 0 byte when named, and nothing in all other cases)
	TAG_Byte   // A single signed byte (8 bits)
	TAG_Short  // A signed short (16 bits, big endian)
	TAG_Int    // A signed short (32 bits, big endian)
	TAG_Long   // A signed long (64 bits, big endian)
	TAG_Float  // A floating point value (32 bits, big endian, IEEE 754-2008, binary32)
	TAG_Double // A floating point value (64 bits, big endian, IEEE 754-2008, binary64)

	// TAG_Int length
	// An array of bytes of unspecified format. The length of this array is <length> bytes
	TAG_Byte_Array

	// TAG_Short length
	// An array of bytes defining a string in UTF-8 format. The length of this array is <length> bytes
	TAG_String

	// TAG_Byte tagId
	// TAG_Int length
	// A sequential list of Tags (not Named Tags), of type <typeId>. The length of this array is <length> Tags
	// Notes:   All tags share the same type.
	TAG_List

	// A sequential list of Named Tags. This array keeps going until a TAG_End is found.
	// TAG_End end
	// Notes:   If there's a nested TAG_Compound within this tag, that one will also have a TAG_End,
	//          so simply reading until the next TAG_End will not work. The names of the named tags
	//			have to be unique within each TAG_Compound
	//          The order of the tags is not guaranteed.
	TAG_Compound
)

var typeToString = map[TagType]string{
	TAG_End:        "TAG_End",
	TAG_Byte:       "TAG_Byte",
	TAG_Short:      "TAG_Short",
	TAG_Int:        "TAG_Int",
	TAG_Long:       "TAG_Long",
	TAG_Float:      "TAG_Float",
	TAG_Double:     "TAG_Double",
	TAG_Byte_Array: "TAG_Byte_Array",
	TAG_String:     "TAG_String",
	TAG_List:       "TAG_List",
	TAG_Compound:   "TAG_Compound",
}

type Tag interface{}
type NamedTag struct {
	tagType TagType
	name    string
	payload Tag
}

func printUnnamedTag(buffer *string, prefix string, tag Tag, tagType TagType) {
	*buffer += prefix + typeToString[tagType]
	if tagType == TAG_Compound {
		elements := tag.([]NamedTag)
		*buffer += fmt.Sprintf(": %d entries\n%s{\n", len(elements), prefix)
		for _, element := range elements {
			element.Print(buffer, "  "+prefix)
		}
		*buffer += prefix + "}\n"
	} else if tagType == TAG_List {
		tagList := tag.(ListTag)
		*buffer += fmt.Sprintf(": %d entries of type %s\n%s{\n", len(tagList.elements), typeToString[tagList.listType], prefix)
		for _, element := range tagList.elements {
			printUnnamedTag(buffer, "  "+prefix, element, tagList.listType)
		}
		*buffer += prefix + "}\n"
	} else {
		*buffer += fmt.Sprintf(": %v\n", tag)
	}
}

func (tag NamedTag) Print(buffer *string, prefix string) {
	*buffer += prefix + typeToString[tag.tagType]
	if tag.tagType == TAG_Compound {
		elements := tag.payload.([]NamedTag)
		*buffer += fmt.Sprintf("(\"%s\"): %d entries\n%s{\n", tag.name, len(elements), prefix)
		for _, element := range elements {
			element.Print(buffer, "  "+prefix)
		}
		*buffer += prefix + "}\n"
	} else if tag.tagType == TAG_List {
		tagList := tag.payload.(ListTag)
		*buffer += fmt.Sprintf("(\"%s\"): %d entries of type %s\n%s{\n", tag.name, len(tagList.elements), typeToString[tagList.listType], prefix)
		for _, element := range tagList.elements {
			printUnnamedTag(buffer, "  "+prefix, element, tagList.listType)
		}
		*buffer += prefix + "}\n"
	} else {
		*buffer += fmt.Sprintf("(\"%s\"): %v\n", tag.name, tag.payload)
	}
}

var tagParseFuncsRef map[TagType](func([]byte) (Tag, uint)) // workaround to avoid initialization loop
var tagParseFuncs = map[TagType](func([]byte) (Tag, uint)){
	TAG_End:   func(payload []byte) (Tag, uint) { return nil, 0 },
	TAG_Byte:  func(payload []byte) (Tag, uint) { return payload[0], 1 },
	TAG_Short: func(payload []byte) (Tag, uint) { return int16(binary.BigEndian.Uint16(payload[0:2])), 2 },
	TAG_Int:   func(payload []byte) (Tag, uint) { return int32(binary.BigEndian.Uint32(payload[0:4])), 4 },
	TAG_Long:  func(payload []byte) (Tag, uint) { return int64(binary.BigEndian.Uint64(payload[0:8])), 8 },
	TAG_Float: func(payload []byte) (Tag, uint) {
		return math.Float32frombits(binary.BigEndian.Uint32(payload[0:4])), 4
	},
	TAG_Double: func(payload []byte) (Tag, uint) {
		return math.Float64frombits(binary.BigEndian.Uint64(payload[0:8])), 8
	},
	TAG_Byte_Array: func(payload []byte) (Tag, uint) {
		length := binary.BigEndian.Uint32(payload[0:4])
		return payload[4 : 4+length], uint(4 + length)
	},
	TAG_String: func(payload []byte) (Tag, uint) {
		length := binary.BigEndian.Uint16(payload[0:2])
		return string(payload[2 : 2+length]), uint(2 + length)
	},
	TAG_Compound: parseCompoundTag,
	TAG_List:     parseListTag,
}

func parseCompoundTag(payload []byte) (Tag, uint) { // Payload: []NamedTag
	var ret []NamedTag
	var current uint
	for {
		tag, length := parseNamedTag(payload[current:])
		current += length
		if tag.tagType == TAG_End {
			break
		}
		ret = append(ret, tag)
	}
	return ret, current
}

type ListTag struct {
	listType TagType
	elements []Tag
}

func parseListTag(payload []byte) (Tag, uint) { // Payload: []NamedTag
	var ret []Tag
	tagType := TagType(payload[0])
	length := binary.BigEndian.Uint32(payload[1:5])
	var current uint = 5
	for i := 0; i < int(length); i++ {
		tag, length := tagParseFuncsRef[tagType](payload[current:])
		current += length
		ret = append(ret, tag)
	}
	return ListTag{tagType, ret}, current
}

func parseNamedTag(data []byte) (NamedTag, uint) {
	// A Named Tag has the following format:
	// byte tagType
	// TAG_String name
	// [payload]
	var namedTag NamedTag
	var nameLength, payloadStart uint16
	namedTag.tagType = TagType(data[0])
	if namedTag.tagType != TAG_End {
		nameLength = binary.BigEndian.Uint16(data[1:3])
		namedTag.name = string(data[3 : 3+nameLength])
		payloadStart = 3 + nameLength
	} else {
		nameLength = 0
		namedTag.name = "" // The name is assumed to be "" in case of TAG_End
		payloadStart = 1
	}
	var payloadLength uint
	namedTag.payload, payloadLength = tagParseFuncsRef[namedTag.tagType](data[payloadStart:])
	return namedTag, uint(uint(payloadStart) + payloadLength)
}
