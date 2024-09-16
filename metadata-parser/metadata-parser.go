package metadata

import (
	"bufio"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/xml"
	"log"
	"os"
	"strings"
)

type PlayerState byte

const (
	PlayerStateStopped PlayerState = 0x00
	PlayerStatePlaying PlayerState = 0x01
	PlayerStatePaused  PlayerState = 0x02
	PlayerStateError   PlayerState = 0xff
)

type Item struct {
	XMLName xml.Name `xml:"item"`
	Type    string   `xml:"type"`
	Code    string   `xml:"code"`
	Length  int      `xml:"length"`
	Data    string   `xml:"data"`
}

type MetadataParser struct {
	IndexStr   string
	TrackIndex int32
	Artist     string
	Album      string
	Title      string
	Length     int
	Status     PlayerState
}

func (mp *MetadataParser) Start() {
	// TODO: make a parameter
	file, err := os.Open("/tmp/shairport-sync-metadata")
	if err != nil {
		log.Fatal(err)
	}
	// defer file.Close()

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 641024)
	scanner.Buffer(buf, 10241024)

	var tag string
	for scanner.Scan() {
		chunk := scanner.Text()
		tag += chunk

		if strings.HasSuffix(chunk, "</item>") {
			var item Item
			if err := xml.Unmarshal([]byte(tag), &item); err != nil {
				log.Printf("Error unmarshaling XML: %v", err)
				continue
			}

			decodedItem := decodeItem(item)
			mp.processItem(decodedItem)
			tag = ""
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

func (mp *MetadataParser) processItem(item Item) {
	switch item.Code {
	case "asar":
		log.Printf("Artist: %s", item.Data)
		mp.Artist = item.Data
	case "asal":
		log.Printf("Album Name: %s", item.Data)
		mp.Album = item.Data
	case "minm":
		log.Printf("Title: %s", item.Data)
		mp.Title = item.Data
	case "astm":
		if len(item.Data) >= 4 {
			trackLength := binary.BigEndian.Uint32([]byte(item.Data))
			log.Printf("Track length: %d", trackLength)
			mp.Length = int(trackLength)
		} else {
			log.Printf("Invalid track length data")
		}
	case "pffr", "pres":
		log.Printf(">> Play")
		mp.Status = PlayerStatePlaying
	case "paus", "pend":
		log.Printf(">> Pause")
		mp.Status = PlayerStatePaused
	case "mdst":
		log.Printf("Metadata bundle start")
	case "mden":
		log.Printf("Metadata bundle end")
		indexStr := mp.Album + mp.Artist + mp.Title
		if indexStr != mp.IndexStr {
			mp.TrackIndex++
			mp.IndexStr = indexStr
		}
	default:
		log.Printf("Unknown code: %s", item.Code)
	}
}

func (mp *MetadataParser) Stop() {
	// Placeholder for any stop functionality required
}

func decodeItem(item Item) Item {
	iCode, _ := hex.DecodeString(item.Code)
	item.Code = string(iCode)

	iType, _ := hex.DecodeString(item.Type)
	item.Type = string(iType)

	iData, _ := base64.StdEncoding.DecodeString(item.Data)
	item.Data = string(iData)

	return item
}
