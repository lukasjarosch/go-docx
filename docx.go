package docx

import (
	"archive/zip"
	"bytes"
	"fmt"
	"github.com/microcosm-cc/bluemonday"
	"io"
	"os"
	"path/filepath"
)

const (
	DocumentXmlPath = "word/document.xml"
)

// Docx is the representation of a docx file.
type Docx struct {
	originalPath string
	files        []*zip.File
	documentXml []byte
	linksXml     string
	headersXml   map[string]string
	footersXml   map[string]string
}

// Content returns the actual content of the docx file (word/document.xml) as string.
func (d *Docx) Content() string {
	return string(d.documentXml)
}

// Bytes returns the actual content of the docx file (word/document.xml) as raw byte slice
func (d *Docx) Bytes() []byte {
	return d.documentXml
}

// Plaintext returns the document with all tags stripped.
func (d *Docx) Plaintext() string {
	return bluemonday.StripTagsPolicy().Sanitize(d.Content())
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

		// currently, we're only rewriting the main document.xml.
		// this is meant to be expanded in the future.
		switch file.Name {
		case DocumentXmlPath:
			if err := write(fileWriter, file.Name, []byte(d.documentXml)); err != nil {
				return err
			}
			break
		default:
			// default case, open the original file which we got from the archive and copy the bytes into the new archive
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
