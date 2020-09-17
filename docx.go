package docx

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"

	"github.com/microcosm-cc/bluemonday"
)

const (
	DocumentXmlPath = "word/document.xml"
)

var (
	HeaderPathRegex = regexp.MustCompile(`word/header[0-9]*.xml`)
	FooterPathRegex = regexp.MustCompile(`word/footer[0-9]*.xml`)
)

// Docx is the representation of a docx file.
type Docx struct {
	originalPath string
	files        []*zip.File
	documentXml  []byte
	headersXml   map[string][]byte
	footersXml   map[string][]byte
}

// Content returns the actual content of the docx file (word/document.xml) as string.
func (d *Docx) Content() string {
	return string(d.documentXml)
}

// DocumentBytes returns the actual content of the docx file (word/document.xml) as raw byte slice
func (d *Docx) DocumentBytes() []byte {
	return d.documentXml
}

func (d *Docx) Headers() map[string][]byte {
	return d.headersXml
}

func (d *Docx) SetHeaders(headers map[string][]byte) {
	d.headersXml = headers
}

func (d *Docx) Footer() map[string][]byte {
	return d.footersXml
}

func (d *Docx) SetFooters(footers map[string][]byte) {
	d.footersXml = footers
}

// Plaintext returns the document with all tags stripped.
func (d *Docx) Plaintext(data []byte) string {
	return bluemonday.StripTagsPolicy().Sanitize(string(data))
}

// SetContent will overwrite the content of the docx file (word/document.xml).
func (d *Docx) SetDocumentContent(new []byte) {
	d.documentXml = new
}

// WriteFile is a convenience wrapper function around Write().
// It will create the given path, making sure the directories along the path are created.
// Once the file is created, the writer is passed along to Write().
//
// WriteFile will also ensure that the given path and the original docx path are NOT equal.
// Writing into the same file again will break and Write() will simply return with an EOF.
func (d *Docx) WriteFile(path string) error {
	if path == d.originalPath {
		return fmt.Errorf("WriteFile cannot write into the original file")
	}

	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return fmt.Errorf("unable to ensure path directories: %s", err)
	}

	target, err := os.Create(path)
	if err != nil {
		return err
	}
	defer target.Close()

	return d.Write(target)
}

// Write is responsible for assembling a new .docx file using the modified data as well as all remaining files.
// Docx files are basically zip archives with many XMLs included. Only some of the XML files are relevant for this
// library, namely the document.xml which contains the actual body of the docx.
// Files which cannot be modified through this lib will just be read from the original docx and copied into the new.
func (d *Docx) Write(writer io.Writer) error {
	zipWriter := zip.NewWriter(writer)
	defer zipWriter.Close()

	for _, file := range d.files {
		log.Print(file.Name)

		fileWriter, err := zipWriter.Create(file.Name)
		if err != nil {
			return fmt.Errorf("unable to create zip writer: %s", err)
		}

		write := func(writer io.Writer, filename string, data []byte) error {
			_, err := writer.Write(data)
			if err != nil && err != io.EOF {
				return fmt.Errorf("unable to write '%s': %s", filename, err)
			}
			return nil
		}

		for fileName, headerBytes := range d.headersXml {
			if file.Name == fileName {
				if err := write(fileWriter, file.Name, headerBytes); err != nil {
					return err
				}
				break
			}
		}

		for fileName, footerBytes := range d.footersXml {
			if file.Name == fileName {
				if err := write(fileWriter, file.Name, footerBytes); err != nil {
					return err
				}
				break
			}
		}

		if file.Name == DocumentXmlPath {
			if err := write(fileWriter, file.Name, d.documentXml); err != nil {
				return err
			}
		}

		// default case, open the original file which we got from the archive and copy the bytes into the new archive
		if file.Name != DocumentXmlPath &&
			!HeaderPathRegex.MatchString(file.Name) &&
			!FooterPathRegex.MatchString(file.Name)	{
			readCloser, err := file.Open()
			if err != nil {
				return fmt.Errorf("unable to open %s: %s", file.Name, err)
			}

			_, err = fileWriter.Write(d.readBytes(readCloser))
			if err != nil {
				return err
			}
			err = readCloser.Close()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// readBytes is a convenience function to slurp all bytes from a given io.Reader and return them.
// In case nothing could be read, an empty byte array is returned.
func (d *Docx) readBytes(stream io.Reader) []byte {
	buf := new(bytes.Buffer)
	n, err := buf.ReadFrom(stream)

	if n == 0 || err != nil {
		return buf.Bytes()
	}
	return buf.Bytes()
}
