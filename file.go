package extract

import (
	"crypto/sha256"
	"fmt"
)

type FileType string

const JPG = FileType("jpg")
const PNG = FileType("png")
const PDF = FileType("pdf")

type File struct {
	Bytes       []byte
	ContentType FileType
	Checksum    string
}

func checksum(bs []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(bs))
}

func NewJPG(bs []byte) *File {
	return &File{
		Bytes:       bs,
		ContentType: JPG,
		Checksum:    checksum(bs),
	}
}

func NewPNG(bs []byte) *File {
	return &File{
		Bytes:       bs,
		ContentType: PNG,
		Checksum:    checksum(bs),
	}
}
func NewPDF(bs []byte) *File {
	return &File{
		Bytes:       bs,
		ContentType: PDF,
		Checksum:    checksum(bs),
	}
}
