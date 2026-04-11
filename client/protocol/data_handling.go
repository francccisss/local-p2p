package protocol

import (
	"client/utils"
	"fmt"
	"math"
	"os"
)

// Payload of RPCMessage for LEECH call method
type DataSegment struct {
	TotalSegments   int
	SegmentPosition int
	SegmentNum      int
	ClusterName     ClusterName // string ID of the file to be sent
	DataChunk       []byte
}

func VerifyChecksum() {

}

func minMax() {

}

// multiple peers would need to coordinate how many segments
func DataSegmentation(buf []byte, numOfSegments int) ([]DataSegment, error) {
	fmt.Printf("Number of segments to create: %d\n", numOfSegments)

	dataLen := len(buf)

	fmt.Printf("Size of data: %d\n", dataLen)
	var tmp []DataSegment

	lenOfSegments := int(dataLen/numOfSegments) + 1 // + 1 for the remaining bytes when division has remainder

	fmt.Printf("Length of a Segment: %d\n", lenOfSegments)
	for i := 0; i < numOfSegments; i++ {

		// segmentIndex increases relative to i
		// if lenOfSegments = 5
		// i = 0 segmentIndex moves 0
		// i = 1 segmentIndex is 5
		// i = 2 segmentIndex is 10...

		// segmentIndex+diff is the range
		// so range moves relative to segmentIndex
		// from segmentIndex move depending on the remaining data
		// in the buf to be pushed into the datasegment
		// if segmentIndex = 0
		// range = segmentIndex + lenOfsegments
		// so range is 5 since segment = 0
		// segmentIndex = 5
		// range = segment + lenOfSegments
		// so range is 5 + 5 = 10
		// difference compares lenOfSegments and remaining data
		// if the calculation of the remaining data > lenOfSegments
		// then add segmentIndex by lenOfSegments else use minimum value

		segmentIndex := i * lenOfSegments
		diff := min(lenOfSegments, int(math.Abs(float64((segmentIndex)-len(buf))))) // i hate this conversion
		ds := DataSegment{
			SegmentNum: max(0, i),
			DataChunk:  buf[segmentIndex : segmentIndex+diff],
		}
		tmp = append(tmp, ds)
	}

	return tmp, nil
}

func Checkfile(hashKey string, FILE_LOCATION string) (os.DirEntry, string, error) {

	programwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error Unable to Read Get Current Working Directory\n")
		fmt.Printf("Reason: %s\n", err)
		return nil, "", err
	}
	wd := []string{programwd}
	wd = append(wd, FILE_LOCATION)

	entries, err := os.ReadDir(utils.ConcatStr(&wd))

	entry, err := recursiveFileSearch(hashKey, entries, &wd)

	if err != nil {
		return nil, "", fmt.Errorf("No entries matching the fileKey.")
	}

	return entry, utils.ConcatStr(&wd), nil
}

// initializes `wd` with the current working directory of the program
// appended with the file location of the user and as `entries` array is iterated
// and if the current entry is a Directory the `wd` is appended with the current name
// of the directory, and if not then continue.
// If the current file is not a directory and matches the `fileKey` the return the entry of that file
func recursiveFileSearch(fileKey string, entries []os.DirEntry, wd *[]string) (os.DirEntry, error) {
	for _, entry := range entries {
		info, err := entry.Info()
		entryName := info.Name()
		fmt.Printf("entry: %s\n", entryName)
		if err != nil {
			fmt.Printf("Error Unable to get info for file: %s\n", entryName)
			fmt.Printf("Reason: %s\n", err)
			continue
		}
		if !info.IsDir() {
			if entryName == fileKey {
				return entry, nil
			}
			continue
		}
		currentDirectory := entryName + "/"
		*wd = append(*wd, currentDirectory)
		curDirEntries, err := os.ReadDir(utils.ConcatStr(wd))
		if err != nil {
			*wd = (*wd)[:len(*wd)-1]
			fmt.Printf("Error Unable to Read from Directory: %s\n", entryName)
			fmt.Printf("Reason: %s\n", err)
			continue
		}
		foundEntry, err := recursiveFileSearch(fileKey, curDirEntries, wd)
		if err != nil {
			*wd = (*wd)[:len(*wd)-1]
			continue
		}
		return foundEntry, nil
	}
	return nil, fmt.Errorf("No entries matching the fileKey.")
}
