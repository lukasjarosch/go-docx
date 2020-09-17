package docx

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
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
	var documentFile *zip.File
	var headers []*zip.File
	var footers []*zip.File

	// extract the files in which we're interested
	for _, f := range a.data.File {
		log.Print(f.Name)
		if f.Name == DocumentXmlPath {
			documentFile = f
		}
		if HeaderPathRegex.MatchString(f.Name) {
			headers = append(headers, f)
		}
		if FooterPathRegex.MatchString(f.Name) {
			footers = append(footers, f)
		}
	}

	if documentFile == nil {
		return nil, fmt.Errorf("invalid docx archive format, %s missing", DocumentXmlPath)
	}

	readZipFile := func(file *zip.File) []byte {
		readCloser, err := file.Open()
		if err != nil {
			return nil
		}
		defer readCloser.Close()
		fileBytes, err := ioutil.ReadAll(readCloser)
		if err != nil {
			return nil
		}
		return fileBytes
	}

	// create Docx struct on which we can perform our modifications
	docx := &Docx{
		originalPath: a.path,
		files:        a.data.File,
		documentXml:  readZipFile(documentFile),
		headersXml: make(map[string][]byte, len(headers)),
		footersXml: make(map[string][]byte, len(footers)),
	}
	for _, header := range headers {
		docx.headersXml[header.Name] = readZipFile(header)
	}
	for _, footer := range footers {
		docx.footersXml[footer.Name] = readZipFile(footer)
	}

	return docx, nil
}
