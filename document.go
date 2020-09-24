package docx

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
)

const (
	DocumentXml = "word/document.xml"
)

var (
	HeaderPathRegex = regexp.MustCompile(`word/header[0-9]*.xml`)
	FooterPathRegex = regexp.MustCompile(`word/footer[0-9]*.xml`)
)

// Document exposes the main API of the library. It represents the actual docx document which is going to be modified.
type Document struct {
	path     string
	docxFile *os.File
	zipFile  *zip.Reader

	// all files from the zip archive which we're interested in
	files FileMap
	// paths to all header files inside the zip archive
	headerFiles []string
	// paths to all footer files inside the zip archive
	footerFiles []string
}

// Open will open and parse the file pointed to by path.
// The file must be a valid docx file or an error is returned.
func Open(path string) (*Document, error) {
	fh, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("unable to open .docx docxFile: %s", err)
	}

	rc, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("unable to open zip reader: %s", err)
	}

	doc := &Document{
		docxFile: fh,
		zipFile:  rc,
		path:     path,
		files:    make(FileMap),
	}

	if err := doc.parseArchive(); err != nil {
		return nil, fmt.Errorf("error parsing document: %s", err)
	}

	// a valid docx document should really contain a document.xml :)
	if _, exists := doc.files[DocumentXml]; !exists {
		return nil, fmt.Errorf("invalid docx archive, %s is missing", DocumentXml)
	}

	return doc, nil
}

func OpenBytes(b []byte) (*Document, error) {
	rc, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))

	if err != nil {
		return nil, fmt.Errorf("unable to open zip reader: %s", err)
	}

	doc := &Document{
		docxFile: nil,
		zipFile:  rc,
		path:     "",
		files:    make(FileMap),
	}

	if err := doc.parseArchive(); err != nil {
		return nil, fmt.Errorf("error parsing document: %s", err)
	}

	// a valid docx document should really contain a document.xml :)
	if _, exists := doc.files[DocumentXml]; !exists {
		return nil, fmt.Errorf("invalid docx archive, %s is missing", DocumentXml)
	}

	return doc, nil
}

// ReplaceAll will iterate over all files and perform the replacement according to the PlaceholderMap.
func (d *Document) ReplaceAll(placeholderMap PlaceholderMap) error {
	for name, fileBytes := range d.files {
		changedBytes, err := d.replace(placeholderMap, fileBytes)
		if err != nil {
			return err
		}

		err = d.SetFile(name, changedBytes)
		if err != nil {
			return err
		}
	}
	return nil
}

// replace will create a parser on the given bytes, execute it and replace every placeholders found with the data
// from the placeholderMap.
func (d *Document) replace(placeholderMap PlaceholderMap, docBytes []byte) ([]byte, error) {
	// parse the document, extracting run and text positions
	parser := NewRunParser(docBytes)
	err := parser.Execute()
	if err != nil {
		return nil, err
	}

	// use the parsed runs to find all placeholders
	placeholders := ParsePlaceholders(parser.Runs(), docBytes)
	replacer := NewReplacer(docBytes, placeholders)

	for key, value := range placeholderMap {
		err := replacer.Replace(key, value.(string))
		if err != nil {
			if errors.Is(err, ErrPlaceholderNotFound) {
				continue
			} else {
				return nil, err
			}
		}
	}

	return replacer.Bytes(), nil
}

// GetFile returns the content of the given fileName if it exists.
func (d *Document) GetFile(fileName string) []byte {
	if f, exists := d.files[fileName]; exists {
		return f
	}
	return nil
}

// SetFile allows setting the file contents of the given file.
// The fileName must be known, otherwise an error is returned.
func (d *Document) SetFile(fileName string, fileBytes []byte) error {
	if _, exists := d.files[fileName]; !exists {
		return fmt.Errorf("unregistered file %s", fileName)
	}
	d.files[fileName] = fileBytes
	return nil
}

// parseArchive will go through the docx zip archive and read them into the FileMap.
// Files inside the FileMap are those which can be modified by the lib.
// Currently not all files are read, only:
// 	- word/document.xml
//	- word/header*.xml
//	- word/footer*.xml
func (d *Document) parseArchive() error {
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

	for _, file := range d.zipFile.File {
		if file.Name == DocumentXml {
			d.files[DocumentXml] = readZipFile(file)
		}
		if HeaderPathRegex.MatchString(file.Name) {
			d.files[file.Name] = readZipFile(file)
			d.headerFiles = append(d.headerFiles, file.Name)
		}
		if FooterPathRegex.MatchString(file.Name) {
			d.files[file.Name] = readZipFile(file)
			d.footerFiles = append(d.footerFiles, file.Name)
		}
	}

	return nil
}

// WriteToFile will write the document to a new file.
// It is important to note that the target file cannot be the same as the path of this document.
// If the path is not yet created, the function will attempt to MkdirAll() before creating the file.
func (d *Document) WriteToFile(file string) error {
	if file == d.path {
		return fmt.Errorf("WriteToFile cannot write into the original docx archive while it's open")
	}

	err := os.MkdirAll(filepath.Dir(file), 0755)
	if err != nil {
		return fmt.Errorf("unable to ensure path directories: %s", err)
	}

	target, err := os.Create(file)
	if err != nil {
		return err
	}
	defer target.Close()

	return d.Write(target)
}

// Write is responsible for assembling a new .docx docxFile using the modified data as well as all remaining files.
// Docx files are basically zip archives with many XMLs included.
// Files which cannot be modified through this lib will just be read from the original docx and copied into the writer.
func (d *Document) Write(writer io.Writer) error {
	zipWriter := zip.NewWriter(writer)
	defer zipWriter.Close()

	// writes all files which can be modified through the Document and returns true if the file was written.
	// If no file was written, false is returned.
	writeModifiedFile := func(writer io.Writer, zipFile *zip.File) (bool, error) {
		for _, headerFile := range d.headerFiles {
			if zipFile.Name == headerFile {
				if err := d.files.Write(writer, zipFile.Name); err != nil {
					return false, fmt.Errorf("unable to writeFile %s: %s", zipFile.Name, err)
				}
				return true, nil
			}
		}
		for _, footerFile := range d.footerFiles {
			if zipFile.Name == footerFile {
				if err := d.files.Write(writer, zipFile.Name); err != nil {
					return false, fmt.Errorf("unable to writeFile %s: %s", zipFile.Name, err)
				}
				return true, nil
			}
		}
		if zipFile.Name == DocumentXml {
			if err := d.files.Write(writer, zipFile.Name); err != nil {
				return false, err
			}
			return true, nil
		}

		return false, nil
	}

	for _, zipFile := range d.zipFile.File {
		fw, err := zipWriter.Create(zipFile.Name)
		if err != nil {
			return fmt.Errorf("unable to create writer: %s", err)
		}

		// if the file is a file which we might have modified (aka is in the FileMap)
		written, err := writeModifiedFile(fw, zipFile)
		if err != nil {
			return err
		}
		if written {
			continue
		}

		// all unwritten files were not modified by us are thus copied from the source docx.
		readCloser, err := zipFile.Open()
		if err != nil {
			return fmt.Errorf("unable to open %s: %s", zipFile.Name, err)
		}
		_, err = fw.Write(readBytes(readCloser))
		if err != nil {
			return fmt.Errorf("unable to writeFile zipFile %s: %s", zipFile.Name, err)
		}
		err = readCloser.Close()
		if err != nil {
			return fmt.Errorf("unable to close reader for %s: %s", zipFile.Name, err)
		}
	}

	return nil
}

// Close will close everything :)
func (d *Document) Close() {
	if d.docxFile != nil {
		err := d.docxFile.Close()
		if err != nil {
			log.Fatal(err)
		}
	}
	err = d.zipFile.Close()
	if err != nil {
		log.Fatal(err)
	}
}

// FileMap is just a convenience type for the map of fileName => fileBytes
type FileMap map[string][]byte

// Write will try to write the bytes from the map into the given writer.
func (fm FileMap) Write(writer io.Writer, filename string) error {
	file, ok := fm[filename]
	if !ok {
		return fmt.Errorf("file not found %s", filename)
	}

	_, err := writer.Write(file)
	if err != nil && err != io.EOF {
		return fmt.Errorf("unable to writeFile '%s': %s", filename, err)
	}
	return nil
}

// readBytes reads an io.Reader into []byte and returns it.
func readBytes(stream io.Reader) []byte {
	buf := new(bytes.Buffer)
	n, err := buf.ReadFrom(stream)

	if n == 0 || err != nil {
		return buf.Bytes()
	}
	return buf.Bytes()
}
