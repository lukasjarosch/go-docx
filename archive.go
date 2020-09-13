package docx

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
)

// An OOXML document is basically a zip archive. This is the representation for DOCX file archives.
type Archive struct {
	data *zip.Reader
	path string
}

// NewArchive will attempt to open the given path with a zip reader.
// The zip reader is open at this point and must be closed by the caller of this function via Close().
func NewByteArchive(reader io.Reader) (*Archive, error) {
	buff := bytes.NewBuffer([]byte{})
	size, err := io.Copy(buff, reader)
	if err != nil {
		return nil, err
	}
	bReader := bytes.NewReader(buff.Bytes())

	zipReader, err := zip.NewReader(bReader, size)
	if err != nil {
		return nil, fmt.Errorf("unable to open docx archive: %s", err)
	}

	a := &Archive{
		path: "",
		data: zipReader,
	}

	return a, nil
}

// Parse will read out the archive to create a Docx struct which we then can modify to our needs.
func (a *Archive) Parse() (*Docx, error) {
	// extract files which we want to modify in Docx
	var documentFile *zip.File
	for _, f := range a.data.File {
		if f.Name == DocumentXmlPath {
			documentFile = f
		}
	}

	// open files which we want to modify
	if documentFile == nil {
		return nil, fmt.Errorf("invalid docx archive format, %s missing", DocumentXmlPath)
	}
	readCloser, err := documentFile.Open()
	if err != nil {
		return nil, fmt.Errorf("unable to open document.xml: %s", err)
	}

	// read
	docBytes, err := ioutil.ReadAll(readCloser)
	if err != nil {
		return nil, fmt.Errorf("unable to ReadAll: %s", err)
	}

	// create Docx struct on which we can perform our modifications
	docx := &Docx{
		originalPath: a.path,
		files:        a.data.File,
		documentXml:  docBytes,
	}

	return docx, nil
}
