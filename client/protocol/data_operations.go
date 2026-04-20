package protocol

import (
	"bytes"
	"client/utils"
	"encoding/binary"
	"fmt"
	"os"
	"syscall"
)

type FileMetaData struct {
	Name      string
	Hash      string
	Size      uint64
	BlockSize int64
}

// used for RPC Message by requester
type FileRequest struct {
	Segments  int64
	Hash      string // should be 16bit string
	Size      int64
	Offset    int64
	BlockSize int64
}

// used for RPC Message by receiver to reply to the iniator
type DataSegment struct {
	TotalSegments   int64       // size in bytes / block
	SegmentPosition int64       // current bytes created / block
	ClusterName     ClusterName // string ID of the file to be sent
}

func ParseSegmentHeader(header []byte) (DataSegment, int) {

	fmt.Printf("headerLen: %d\n", len(header))
	hashLen := binary.LittleEndian.Uint32(header[:4])
	fmt.Printf("HASH LEN: %d\n", hashLen)
	return DataSegment{}, 0
	// hash := string(header[3:hashLen])
	// segmentCount := binary.LittleEndian.Uint64(header[4+hashLen : hashLen+12]) // 4 + 8 = 12 bytes
	// segmentPos := binary.LittleEndian.Uint64(header[12+hashLen : hashLen+20])
	return DataSegment{
		ClusterName: ClusterName(""),
		// SegmentPosition: int64(segmentPos),
		// TotalSegments:   int64(segmentCount),
	}, int(hashLen) + 20
}

// Wrapper for ReadFileBuf to return Buffer instead of DataSegment
// Header Spec returns DataSegment bytes:
// 4 bytes size of Hash + 8 bytes TotalSegments + 8 bytes SegmentPosition = 20 byte header
// Variable Length data
// Hash and DataChunk
// DataChunk buffer could be variable sized chunk
func CreateDataSegment(path string, dataInfo *FileRequest) (*bytes.Buffer, int, error) {

	buf, err := ReadFileBuf(path, dataInfo)
	if err != nil {
		return nil, 0, err
	}

	// ClusterName:     ClusterName(dataInfo.Hash),
	// TotalSegments:   dataInfo.Size,
	// SegmentPosition: dataInfo.Offset / dataInfo.BlockSize,

	var segmentBuffer bytes.Buffer
	hashLen := len(dataInfo.Hash)
	segmentBuffer.Grow(hashLen + 20 + len(buf))

	intBuf := make([]byte, 8)

	hashLenBuf, _ := binary.Append(nil, binary.BigEndian, hashLen)
	fmt.Printf("BUF INT VAL: %d, HASH LEN: %d\n", hashLen, binary.LittleEndian.Uint32(hashLenBuf))
	segmentBuffer.Write(hashLenBuf)
	segmentBuffer.Write([]byte(dataInfo.Hash))

	binary.Append(intBuf, binary.BigEndian, dataInfo.Size)

	fmt.Printf("BUF INT VAL: %d, Data Info Size: %d\n", hashLen, binary.LittleEndian.Uint64(intBuf))
	segmentBuffer.Write(intBuf)
	clear(intBuf)

	binary.Append(intBuf, binary.BigEndian, dataInfo.Offset/dataInfo.BlockSize)
	fmt.Printf("BUF INT VAL: %d, Data Info Position: %d\n", hashLen, binary.LittleEndian.Uint64(intBuf))
	segmentBuffer.Write(intBuf)

	headerSize := segmentBuffer.Len()
	segmentBuffer.Write(buf)

	return &segmentBuffer, headerSize, nil

}

// multiple peers would need to coordinate how many segments
func ReadFileBuf(path string, dataInfo *FileRequest) ([]byte, error) {

	fd, err := syscall.Open(path, syscall.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}

	buf, err := syscall.Mmap(fd, dataInfo.Offset, int(dataInfo.BlockSize), syscall.PROT_READ, syscall.MAP_SHARED)

	if err != nil {
		fmt.Println("Error accessing memory mapped disk")
		return nil, err
	}
	rem := min(dataInfo.BlockSize, dataInfo.Size-dataInfo.Offset)
	tmp := make([]byte, rem)
	copy(tmp, buf[:rem])

	err = syscall.Munmap(buf)
	if err != nil {
		fmt.Println("Error accessing memory mapped disk")
		return nil, err
	}

	dataInfo.Offset += int64(dataInfo.BlockSize)
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
		fmt.Printf("Entry located: %s\n", entryName)
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

		fmt.Printf("Found File: %s\n", entryName)
		return foundEntry, nil
	}
	return nil, fmt.Errorf("No entries matching the fileKey.")
}
