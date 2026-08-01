package comicinfo

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	"github.com/mholt/archives"
)

// utf8BOM is the byte-order mark for UTF-8 encoded files.
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// Parse parses ComicInfo XML data into a ComicInfo struct.
// Handles UTF-8 BOM if present.
func Parse(xmlData []byte) (*ComicInfo, error) {
	data := bytes.TrimPrefix(xmlData, utf8BOM)

	var ci ComicInfo
	if err := xml.Unmarshal(data, &ci); err != nil {
		return nil, fmt.Errorf("failed to parse ComicInfo XML: %w", err)
	}
	return &ci, nil
}

// ExtractFromArchive opens a comic archive and extracts ComicInfo.xml if present.
// Returns nil, nil if no ComicInfo.xml found (not an error).
// Supports .cbz/.zip (via archive/zip) and .cbr/.rar/.cb7/.7z (via github.com/mholt/archives).
func ExtractFromArchive(archivePath string) (*ComicInfo, error) {
	ext := strings.ToLower(filepath.Ext(archivePath))
	switch ext {
	case ".cbz", ".zip":
		return extractFromZip(archivePath)
	case ".cbr", ".rar", ".cb7", ".7z":
		return extractFromArchives(archivePath)
	default:
		return nil, fmt.Errorf("unsupported archive format: %s", ext)
	}
}

// extractFromZip extracts ComicInfo.xml from a ZIP/CBZ archive using archive/zip.
func extractFromZip(archivePath string) (*ComicInfo, error) {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open zip archive: %w", err)
	}
	defer r.Close()

	var best *zip.File
	bestDepth := -1

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if !strings.EqualFold(filepath.Base(f.Name), "comicinfo.xml") {
			continue
		}
		depth := strings.Count(f.Name, "/")
		if best == nil || depth < bestDepth {
			best = f
			bestDepth = depth
		}
	}

	if best == nil {
		return nil, nil
	}

	rc, err := best.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open ComicInfo.xml in archive: %w", err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("failed to read ComicInfo.xml: %w", err)
	}

	return Parse(data)
}

// extractFromArchives extracts ComicInfo.xml from RAR/7Z archives using github.com/mholt/archives.
func extractFromArchives(archivePath string) (*ComicInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fsys, err := archives.FileSystem(ctx, archivePath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open archive: %w", err)
	}

	var bestPath string
	bestDepth := -1

	walkErr := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.EqualFold(filepath.Base(path), "comicinfo.xml") {
			return nil
		}
		depth := strings.Count(path, "/")
		if bestPath == "" || depth < bestDepth {
			bestPath = path
			bestDepth = depth
		}
		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("failed to walk archive: %w", walkErr)
	}

	if bestPath == "" {
		return nil, nil
	}

	f, err := fsys.Open(bestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open ComicInfo.xml in archive: %w", err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read ComicInfo.xml: %w", err)
	}

	return Parse(data)
}
