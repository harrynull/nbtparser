package nbtparser

import (
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"
	"log"
)

func decompress(w io.Writer, data []byte) error {
	gr, err := gzip.NewReader(bytes.NewBuffer(data))
	defer gr.Close()
	data, err = ioutil.ReadAll(gr)
	if err != nil {
		return err
	}
	w.Write(data)
	return nil
}

// ParseNBT Parse a NBT
func ParseNBT(data []byte, isCompressed bool) NamedTag {
	tagParseFuncsRef = tagParseFuncs
	if isCompressed {
		var decompressed bytes.Buffer
		err := decompress(&decompressed, data)
		if err != nil {
			log.Fatal("Failed to decompress data: ", err)
		}
		tag, _ := parseNamedTag(decompressed.Bytes())
		return tag
	}
	tag, _ := parseNamedTag(data)
	return tag
}
