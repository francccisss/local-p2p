package protocol

import (
	"bytes"
	"client/utils"
	"encoding/binary"
	"fmt"
	"math"
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
	Pieces    int64
	Hash      string // should be 16bit string
	Size      int64
	Offset    int64
	BlockSize int64
}

// used for RPC Message by receiver to reply to the iniator
type PieceHeader struct {
	PieceSize     int64       // from blockSize or remaining bytes
	PiecePosition int64       // current bytes created / block
	TotalPieces   int64       // size in bytes / block
	ClusterName   ClusterName // string ID of the file to be sent
}

// HashLen, Hash(variable sized), Piece Pos, PieceOffset, total Pieces
func ParsePieceHeader(header []byte) (PieceHeader, int, error) {

	type Header struct {
		PieceSize     int64 // from blockSize or remaining bytes
		PiecePosition int64 // current bytes created / block
		TotalPieces   int64 // size in bytes / block
	}

	fmt.Printf("Length of header: %d\n", len(header))
	hashLen := binary.LittleEndian.Uint32(header[:4])
	fmt.Printf("HASH LEN: %d\n", hashLen)

	fmt.Printf("Hash %s\n", string(header[4:4+hashLen]))

	var h Header
	hreader := bytes.NewReader(header[4+hashLen:]) // dont include string ClusterName
	fmt.Printf("header len remaining: %d", hreader.Len())

	err := binary.Read(hreader, binary.LittleEndian, &h)

	if err != nil {
		return PieceHeader{}, 0, err
	}

	return PieceHeader{
		PieceSize:     h.PieceSize,
		PiecePosition: h.PiecePosition,
		TotalPieces:   h.TotalPieces,
		ClusterName:   ClusterName(header[4 : 4+hashLen]),
	}, len(header), nil
}

// Wrapper for ReadFileBuf to return Buffer instead of PieceHeader
// Header Spec returns PieceHeader bytes:
// 4 bytes size of Hash + 8 bytes TotalPieces + 8 bytes PiecePosition + 8 total Pieces= 20 byte header
// Variable Length data
// Hash and DataChunk
// DataChunk buffer could be variable sized chunk
func CreateDataPiece(path string, dataInfo *FileRequest) (*bytes.Buffer, int, error) {

	fmt.Printf("Creating Pieces for %s\n", dataInfo.Hash)
	buf, err := ReadFileBuf(path, dataInfo)
	if err != nil {
		return nil, 0, err
	}

	PieceBuffer := new(bytes.Buffer)

	hashLen := len(dataInfo.Hash)
	PieceBuffer.Grow(hashLen + 28 + len(buf))

	hashLenBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(hashLenBuf, uint32(hashLen))

	binary.Write(PieceBuffer, binary.LittleEndian, uint32(hashLen))
	PieceBuffer.Write([]byte(dataInfo.Hash))
	fmt.Printf("BUF INT VAL: %d, HASH LEN in Bytes: B>%d, String %s\n", hashLen, hashLenBuf, PieceBuffer.Bytes()[:hashLen])

	// 8 bytes Piece Size
	binary.Write(PieceBuffer, binary.LittleEndian, uint64(len(buf)))
	fmt.Printf("BUF INT VAL: %d, Data Info Piece Size: %d\n", len(buf), binary.LittleEndian.Uint64(PieceBuffer.Bytes()[hashLen+4:]))

	// 8 bytes Offset
	binary.Write(PieceBuffer, binary.LittleEndian, uint64(dataInfo.Offset/dataInfo.BlockSize)) // get current Piece
	fmt.Printf("BUF INT VAL: %d, Data Info Offset: %d\n", dataInfo.Offset, binary.LittleEndian.Uint64(PieceBuffer.Bytes()[hashLen+12:]))

	// 8 bytes TotalPieces
	binary.Write(PieceBuffer, binary.LittleEndian, uint64(math.Ceil(float64(dataInfo.Size)/float64(dataInfo.BlockSize))))
	fmt.Printf("BUF INT VAL: %d, Data Info Total Pieces: %d\n", dataInfo.Size, binary.LittleEndian.Uint64(PieceBuffer.Bytes()[hashLen+12+8:]))

	headerSize := PieceBuffer.Len()
	PieceBuffer.Write(buf)

	return PieceBuffer, headerSize, nil

}

// multiple peers would need to coordinate how many Pieces
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

func Checkfile(fileHash FileHash, FILE_LOCATION string) (os.DirEntry, string, error) {

	programwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error Unable to Read Get Current Working Directory\n")
		fmt.Printf("Reason: %s\n", err)
		return nil, "", err
	}
	wd := []string{programwd}
	wd = append(wd, FILE_LOCATION)

	entries, err := os.ReadDir(utils.ConcatStr(&wd))

	entry, err := recursiveFileSearch(fileHash, entries, &wd)

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

func recursiveFileSearch(fileHash FileHash, entries []os.DirEntry, wd *[]string) (os.DirEntry, error) {
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
			if entryName == fileHash {
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
		foundEntry, err := recursiveFileSearch(fileHash, curDirEntries, wd)
		if err != nil {
			*wd = (*wd)[:len(*wd)-1]
			continue
		}

		fmt.Printf("Found File: %s\n", entryName)
		return foundEntry, nil
	}
	return nil, fmt.Errorf("No entries matching the fileKey.")
}
