package protocol

import (
	"client/utils"
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
	Hash      string
	Size      int64
	Offset    int64
	BlockSize int64
}

// used for RPC Message by receiver to reply to the iniator
type DataSegment struct {
	TotalSegments   int64
	SegmentPosition uint64
	ClusterName     ClusterName // string ID of the file to be sent
	DataChunk       []byte
}

func CreateDataSegment(fd int, i int64, dataInfo FileRequest) (DataSegment, error) {

	buf, err := ReadFileBuf(fd, dataInfo.Size, dataInfo.Offset, dataInfo.BlockSize)
	if err != nil {
		return DataSegment{}, err
	}

	return DataSegment{
		SegmentPosition: uint64(i),
		DataChunk:       buf,
	}, nil

}

// multiple peers would need to coordinate how many segments
func ReadFileBuf(fd int, fileLen int64, offset int64, blockSize int64) ([]byte, error) {

	buf, err := syscall.Mmap(fd, offset, int(blockSize), syscall.PROT_READ, syscall.MAP_SHARED)

	if err != nil {
		fmt.Println("Error accessing memory mapped disk")
		return nil, err
	}
	rem := min(blockSize, fileLen-offset)
	tmp := make([]byte, rem)
	copy(tmp, buf[:rem])

	err = syscall.Munmap(buf)
	if err != nil {
		fmt.Println("Error accessing memory mapped disk")
		return nil, err
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
