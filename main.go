package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"time"
)

var filename string

func main() {

	flag.StringVar(&filename, "f", "", "filename or path/to/file to open")

	flag.Parse()
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalf("failed to open %s, err: %s\n", filename, err)
	}
	walkFile(b)
}

// VideoFile is the most outer box if the mo4 file.
// All other boxes are inside this struct
type VideoFile struct {
	// Type could be absent on old files, in those cases:
	// Files with no file‐type box should be read as if they contained an FTYP box with
	// Major_brand='mp41', minor_version=0, and the single compatible brand'mp41'.
	Type             Atom
	MajorBrand       string
	MinorVersion     int64
	CompatibleBrands []string
	MDATPos          int64
	MVHD             MVHD
	// timescale is always 4 bytes long, ver 0 and 1
	Timescale int64
	Duration  int64
	Rate      int64
	Volume    int64
	Matrix    [9][]byte
}

// PrettyPrint would print the data in a nice format, also display times in local zones
func (v *VideoFile) PrettyPrint() string {
	// TODO: Implement
	ret := fmt.Sprintf("\n%+v\n", v.Type)
	for idx, v := range v.Matrix {
		ret = ret + fmt.Sprintf("Matrix[%d]: % 02X\n", idx, v)
	}
	return ret
}

// Atom is the name and length of each box/section of metadata
type Atom struct {
	Name   string
	Length int64
}

// MVHD atom
type MVHD struct {
	Atom
	// Version if version is 1, then wordLengths are 8 bytes long
	// else, 4 bytes long
	Version int64
	// CreatedOn is set in the file as seconds since midnight, Jan. 1, 1904, in UTC time
	CreatedOn time.Time
	// UpdatedOn is set in the file as seconds since midnight, Jan. 1, 1904, in UTC time
	UpdatedOn time.Time
}

var file VideoFile

func walkFile(b []byte) {
	currStart := int64(0)
	ftypLen, ftyp := getAtomSizeName(b, currStart)
	if ftyp == "ftyp" {
		file.Type = Atom{
			Name:   "ftyp",
			Length: ftypLen,
		}

		ret := getPortion(b, currStart, ftypLen, true)
		currStart += 8
		file.MajorBrand = string(ret[currStart : currStart+4])
		currStart += 4
		file.MinorVersion = byteToI(ret[currStart : currStart+4])
		// move currStart cursor position inside the for loop initiation
		// section (first for loop parameter)
		for currStart += 4; currStart < file.Type.Length; currStart += 4 {
			compBrand := getPortion(b, currStart, currStart+4, true)
			file.CompatibleBrands = append(file.CompatibleBrands, string(compBrand))
		}
	}
	len, name := getAtomSizeName(b, currStart)
	if name == "mdat" {
		// len 1 means we need to read the next 8 bytes (64bit field length) to get the
		// offset of where to start reading the mvhd atom
		if len == 1 {
			// increase by field length and name
			currStart += 8
			file.MDATPos = byteToI(getPortion(b, currStart, currStart+8, true)) // 8 instead of 4 because this is a 64bit word
			// we have an offset, so add up the current position to make it an abolute position
			file.MDATPos += currStart
			currStart = file.MDATPos
			file.MVHD.Length, file.MVHD.Name = getAtomSizeName(b, currStart)
			currStart += 8
			file.MVHD.Version = byteToI(getPortion(b, currStart, currStart+4, true))
			currStart += 4
			wordLength := int64(4)
			if file.MVHD.Version == 1 {
				wordLength = 8
			}

			file.MVHD.CreatedOn = printDateTime(b, currStart, currStart+wordLength)
			currStart += wordLength
			file.MVHD.UpdatedOn = printDateTime(b, currStart, currStart+wordLength)
			currStart += wordLength

			////

			file.Timescale = byteToI(getPortion(b, currStart, currStart+4, true))
			currStart += 4 // not wordLength because timescale is always 4 bytes
			duration := byteToI(getPortion(b, currStart, currStart+wordLength, true))
			// TODO: Add this to prettyPrint?
			log.Printf("duration: '%d' seconds\n", duration/file.Timescale)
			currStart += wordLength
			// 32bits/4 byte word
			file.Rate = byteToI(getPortion(b, currStart, currStart+4, true))
			currStart += 4
			// 16bits/2 bytes word
			file.Volume = byteToI(getPortion(b, currStart, currStart+2, true))
			currStart += 2
			// reserved := getPortion(b, currStart, currStart+2, false) // 16bits/2 byte word
			currStart += 2
			//reserved2 := getPortion(b, currStart, currStart+4+4, false) // array of two 4 byte words
			currStart += 4 + 4
			for x := 0; x < 9; x++ {
				file.Matrix[x] = getPortion(b, currStart, currStart+4, true) // array of 9 4 byte words
				currStart += 4
			}
			// for x := 0; x < 6; x++ {
			// 	preDefined := getPortion(b, currStart, currStart+4, false) // array of 6 4 byte words
			// 	log.Printf("preDefined[%d]: % 02X\n", x, preDefined)
			// 	log.Println("==================")
			// 	currStart += 4
			// }
			// nextTrackID := getPortion(b, currStart, currStart+4, false) // 32bits/4 byte word
			// log.Printf("nextTrackID: % 02X\n", nextTrackID)
			// log.Println("==================")
			// currStart += 4 // end of nextTrackID
			//
			// currStart = findTrckData(b, currStart, wordLength, timeScale, loc)
			// currStart += 4
			// // skipping ‘mdhd’ and others
			// currStart = findCo64Data(b, currStart, wordLength, timeScale, loc)
			//
			// // now we skip some stuff, hopefully it's ok
			// // in future versions we may want to keep track of the path we are going
			// // through, so we are in the "right" field (where same field is multiple times)
			// // This finds the video track handler
			//
			// currStart = findTrckData(b, currStart, wordLength, timeScale, loc)
			// currStart += 4
			// log.Println("+++++++++++++++++++++++++++")
			// currStart = findCo64Data(b, currStart, wordLength, timeScale, loc)

		}
	}

	log.Printf("file is %+v\n", file.PrettyPrint())

	return
	// start reading at position 0x20, which is the 8 bytes (2 sets of 4 bytes) offset
	// from http://xhelmboyx.tripod.com/formats/mp4-layout.txt
	// -> 8 bytes wider mdat box offset = 64-bit unsigned offset
	//   - only if mdat standard offset set to 1
	initialOffset := int64(0x20)
	// read 8 bytes
	var start, end int64 = startEnd(initialOffset)
	n := getPortion(b, start, end, false)
	log.Printf("at position: '%#02X', got: '% 02X'\n", start, n)
	// ret := fmt.Sprintf("%X", n)
	currStart = byteToI(n)
	log.Printf("================== %X\n", currStart)
	s, e := startEnd(currStart + initialOffset)
	a := getPortion(b, s, e, false) //exploring
	//a := getPortion(b, newStart+0X19, newStart+0X19+12) //exploring
	log.Printf("Rount 2: got: '% 02X'\n", a)
	// get box version to see if we use 32 or 64 bit for info (4 bytes or 8 bytes words)
	currStart = s + 8
	// v format version, used in diff places of the parser
	v := byteToI(getPortion(b, currStart, currStart+4, false))
	wordLength := int64(4)
	if v == 1 {
		wordLength = 8
	}
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		log.Fatalln("failed to set New York as timezone", err)
	}

	// get creation time
	// previous start + 8 bytes of data that represent lmvhd + 4 for version
	// then read only 4 bytes for creation time
	currStart += wordLength
	creation := printDateTime(b, currStart, currStart+wordLength)
	log.Println("creation time: ", creation.In(loc))
	currStart += wordLength
	update := printDateTime(b, currStart, currStart+wordLength)
	log.Println("update time:   ", update.In(loc))
	log.Println("==================")
	// timescale is	 an	 integer	 that	 specifies	 the	 time‐scale	 for	 the	 entire	 presentation;	 this	 is	 the
	// number	 of	 time	 units	 that	 pass	 in	 one	 second.	 For	 example,	 a	 time	 coordinate	 system	 that
	// measures	time	in	sixtieths	of	a	second	has	a	time	scale	of	60.
	currStart += wordLength
	timeScale := byteToI(getPortion(b, currStart, currStart+4, false)) // timescale is always 4 bytes long, ver 0 and 1
	log.Printf("timescale: %d\n", timeScale)
	log.Println("==================")
	currStart += 4 // not wordLength because timescale is always 4 bytes
	duration := byteToI(getPortion(b, currStart, currStart+wordLength, false))
	log.Printf("raw duration: %d\n", duration)
	log.Printf("duration: '%d' seconds\n", duration/timeScale)
	log.Println("==================")
	currStart += wordLength
	rate := getPortion(b, currStart, currStart+4, false) // 32bits/4 byte word
	log.Printf("rate: % 02X\n", rate)
	log.Printf("rate(d): %d\n", byteToI(rate))
	log.Println("==================")
	currStart += 4
	volume := getPortion(b, currStart, currStart+2, false) // 16bits/2 byte word
	log.Printf("volume: % 02X\n", volume)
	log.Println("==================")
	currStart += 2
	reserved := getPortion(b, currStart, currStart+2, false) // 16bits/2 byte word
	log.Printf("reserved: % 02X\n", reserved)
	log.Println("==================")
	currStart += 2
	reserved2 := getPortion(b, currStart, currStart+4+4, false) // array of two 4 byte words
	log.Printf("reserved2: % 02X\n", reserved2)
	log.Println("==================")
	currStart += 4 + 4
	for x := 0; x < 9; x++ {
		matrix := getPortion(b, currStart, currStart+4, false) // array of 9 4 byte words
		log.Printf("matrix[%d]: % 02X\n", x, matrix)
		log.Println("==================")
		currStart += 4
	}
	for x := 0; x < 6; x++ {
		preDefined := getPortion(b, currStart, currStart+4, false) // array of 6 4 byte words
		log.Printf("preDefined[%d]: % 02X\n", x, preDefined)
		log.Println("==================")
		currStart += 4
	}
	nextTrackID := getPortion(b, currStart, currStart+4, false) // 32bits/4 byte word
	log.Printf("nextTrackID: % 02X\n", nextTrackID)
	log.Println("==================")
	currStart += 4 // end of nextTrackID

	currStart = findTrckData(b, currStart, wordLength, timeScale, loc)
	currStart += 4
	// skipping ‘mdhd’ and others
	currStart = findCo64Data(b, currStart, wordLength, timeScale, loc)

	// now we skip some stuff, hopefully it's ok
	// in future versions we may want to keep track of the path we are going
	// through, so we are in the "right" field (where same field is multiple times)
	// This finds the video track handler

	currStart = findTrckData(b, currStart, wordLength, timeScale, loc)
	currStart += 4
	log.Println("+++++++++++++++++++++++++++")
	currStart = findCo64Data(b, currStart, wordLength, timeScale, loc)

}

func findTrckData(b []byte, currStart, wordLength, timeScale int64, loc *time.Location) int64 {
	// now we skip some stuff, hopefully it's ok
	// in future versions we may want to keep track of the path we are going
	// through, so we are in the "right" field (where same field is multiple times)
	for {
		tkhd := getPortion(b, currStart, currStart+4, true) // 32bits/4 byte word
		if "tkhd" == string(tkhd[:]) {
			log.Printf("tkhd: % 02X\n", tkhd)
			log.Println("==================")
			break
		}
		// increase only by one byte because somewhere in the
		// middle, there is a value or something that is just one byte.
		currStart++
	}
	currStart += 4

	trackFlags := getPortion(b, currStart, currStart+4, false) // 32bits/4 byte word
	log.Printf("trackVersion: % 02X\n", trackFlags)
	log.Println("==================")
	currStart += 4
	tCreation := printDateTime(b, currStart, currStart+wordLength)
	log.Println("tCreation time: ", tCreation.In(loc))
	currStart += wordLength
	tUpdate := printDateTime(b, currStart, currStart+wordLength)
	log.Println("tUpdate time:   ", tUpdate.In(loc))
	log.Println("==================")
	currStart += wordLength
	trackID := getPortion(b, currStart, currStart+4, false) // 32bits/4 byte word
	log.Printf("trackID: % 02X\n", trackID)
	log.Println("==================")
	currStart += 4
	// reserved 4 byte field
	currStart += 4
	tDuration := byteToI(getPortion(b, currStart, currStart+wordLength, true))
	log.Printf("raw duration: %d\n", tDuration)
	log.Printf("duration: '%d' seconds\n", tDuration/timeScale)
	log.Println("==================")
	currStart += wordLength
	// reserved array of 2 4 byte fields
	currStart += 4
	currStart += 4
	layer := getPortion(b, currStart, currStart+2, false) // 16bits/2 byte word
	log.Printf("layer: % 02X\n", layer)
	log.Println("==================")
	currStart += 2
	alternateGroup := getPortion(b, currStart, currStart+2, false) // 16bits/2 byte word
	log.Printf("alternateGroup: % 02X\n", alternateGroup)
	log.Println("==================")
	currStart += 2
	tVolume := getPortion(b, currStart, currStart+2, false) // 16bits/2 byte word
	log.Printf("tVolume: % 02X\n", tVolume)
	log.Println("==================")
	currStart += 2
	currStart += 2
	for x := 0; x < 9; x++ {
		tMatrix := getPortion(b, currStart, currStart+4, false) // array of 9 4 byte words
		log.Printf("tMatrix[%d]: % 02X\n", x, tMatrix)
		log.Println("==================")
		currStart += 4
	}
	width := getPortion(b, currStart, currStart+4, false) // 32bits/4 byte word
	log.Printf("width: % 02X\n", width)
	log.Printf("width: %d\n", byteToI(width[0:2]))
	log.Println("==================")
	currStart += 4
	height := getPortion(b, currStart, currStart+4, false) // 32bits/4 byte word
	log.Printf("height: % 02X\n", height)
	// the field is a 16.16 fixed point, so we split the 4 bytes into two
	// and only read the left side/first 2 bytes
	log.Printf("height: %d\n", byteToI(height[0:2]))
	log.Println("==================")
	return currStart
}

// this is the box that has offset to the actual movie data, packets here are 64 bit long (8 bytes)
// there is also stco, which is 32 bit
func findCo64Data(b []byte, currStart, wordLength, timeScale int64, loc *time.Location) int64 {
	for {
		co64 := getPortion(b, currStart, currStart+4, true) // 32bits/4 byte word
		if "co64" == string(co64[:]) {
			log.Printf("co64: % 02X\n", co64)
			log.Println("==================")
			break
		}
		// increase only by one byte because somewhere in the
		// middle, there is a value or something that is just one byte.
		currStart++
	}
	currStart += 4
	version := getPortion(b, currStart, currStart+4, false) // 32bits/4 byte word
	log.Printf("version: % 02X\n", version)
	log.Println("==================")
	currStart += 4
	entryCnt := byteToI(getPortion(b, currStart, currStart+4, false)) // 32bits/4 byte word
	log.Printf("entryCnt : %d\n", entryCnt)
	log.Println("==================")
	currStart += 4
	// prevChunkOffset := int64(0)

	for x := int64(0); x < entryCnt; x++ {
		chunkOffset := byteToI(getPortion(b, currStart, currStart+8, true)) // array of entryCnt 8 byte words
		log.Printf("chunkOffset[%d]:hex %#X\n", x, chunkOffset)
		// log.Printf("chunkOffset[%d]:dec %d\n", x, chunkOffset)
		// log.Printf("diff:dec  ------------------->>>> %d\n", chunkOffset-prevChunkOffset)
		// prevChunkOffset = chunkOffset
		log.Println("==================")
		separator := getPortion(b, chunkOffset, chunkOffset+5, true)
		log.Printf("separator:hex: %#X <<<========\n", separator)
		// log.Printf("separator:dec: %d <<<xxxxxxxxxx\n", byteToI(separator))
		currStart += 8
	}
	return currStart
}

func getPortion(b []byte, start, end int64, silent bool) []byte {
	if !silent {
		//log.Printf("start is %d\n", start)
		//log.Printf("end   is %d\n", end)
		log.Printf("start is % #X\n", start)
		log.Printf("end   is % #X\n", end)
		log.Printf("ascii:   '%s'\n", b[start:end])
		log.Printf("hex:     % 02X\n", b[start:end])
		log.Printf("offset:  % 02x", start)
	}
	return b[start:end]
}

func startEnd(x int64) (int64, int64) {
	return x, x + 8
}

func printDateTime(b []byte, start, end int64) time.Time {
	log.Printf("===> Time hex:     % 02X\n", b[start:end])
	// midnight,	Jan.	1,	1904,	in	UTC	time
	startingVideoEpoc := time.Date(1904, 1, 1, 0, 0, 0, 0, time.UTC)
	return startingVideoEpoc.Add(time.Duration(byteToI(b[start:end])) * time.Second)
}

func byteToI(b []byte) int64 {
	newStart, err := strconv.ParseInt(fmt.Sprintf("%X", b), 16, 64)
	if err != nil {
		log.Fatalf("failed to convert %+v to int, err: %s\n", b, err)
	}
	return newStart
}
func byteToI16Bit(b []byte) int64 {
	newStart, err := strconv.ParseInt(fmt.Sprintf("%X", b), 16, 64)
	if err != nil {
		log.Fatalf("failed to convert %+v to int, err: %s\n", b, err)
	}
	return newStart
}

func getAtomSizeName(b []byte, start int64) (int64, string) {
	if int64(len(b)) > start+8 {
		return byteToI(b[start : start+4]), string(b[start+4 : start+8])
	}
	return -1, ""
}
