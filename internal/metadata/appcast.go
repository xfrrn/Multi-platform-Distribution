package metadata

import (
	"encoding/xml"
	"path"
	"strings"
	"time"

	"multi-platform-distribution/internal/domain"
)

type Appcast struct {
	XMLName xml.Name       `xml:"rss"`
	Version string         `xml:"version,attr"`
	Sparkle string         `xml:"xmlns:sparkle,attr"`
	Channel AppcastChannel `xml:"channel"`
}

type AppcastChannel struct {
	Title       string        `xml:"title"`
	Description string        `xml:"description"`
	Items       []AppcastItem `xml:"item"`
}

type AppcastItem struct {
	Title       string           `xml:"title"`
	PubDate     string           `xml:"pubDate"`
	Description string           `xml:"description,omitempty"`
	Enclosure   AppcastEnclosure `xml:"enclosure"`
}

type AppcastEnclosure struct {
	URL                string `xml:"url,attr"`
	Length             int64  `xml:"length,attr"`
	Type               string `xml:"type,attr"`
	Version            string `xml:"sparkle:version,attr"`
	ShortVersionString string `xml:"sparkle:shortVersionString,attr"`
}

func RenderAppcast(appSlug string, manifest domain.UpdateManifest) ([]byte, error) {
	items := make([]AppcastItem, 0, len(manifest.Files))
	for _, file := range manifest.Files {
		items = append(items, AppcastItem{
			Title:       manifest.Version + " " + file.Platform + "/" + file.Arch,
			PubDate:     manifest.PublishedAt.UTC().Format(time.RFC1123Z),
			Description: manifest.Changelog,
			Enclosure: AppcastEnclosure{
				URL:                file.URL,
				Length:             file.Size,
				Type:               contentType(file.Type),
				Version:            manifest.Version,
				ShortVersionString: manifest.Version,
			},
		})
	}

	appcast := Appcast{
		Version: "2.0",
		Sparkle: "http://www.andymatuschak.org/xml-namespaces/sparkle",
		Channel: AppcastChannel{
			Title:       appSlug + " updates",
			Description: "Update feed for " + appSlug,
			Items:       items,
		},
	}

	body, err := xml.MarshalIndent(appcast, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), body...), nil
}

func contentType(fileType string) string {
	switch strings.ToLower(strings.TrimPrefix(fileType, ".")) {
	case "dmg":
		return "application/x-apple-diskimage"
	case "zip":
		return "application/zip"
	case "exe":
		return "application/vnd.microsoft.portable-executable"
	case "msi":
		return "application/x-msi"
	case "appimage":
		return "application/x-executable"
	default:
		ext := strings.ToLower(path.Ext("file." + fileType))
		if ext == ".xml" {
			return "application/xml"
		}
		return "application/octet-stream"
	}
}
