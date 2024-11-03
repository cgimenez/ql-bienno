package main

import (
	"fmt"
	"path"
	"regexp"
	"strings"
)

type ArchInfo struct {
	OS      string
	Arch    string
	Version string
}

type StockImage struct {
	Name     string
	URL      string
	ArchInfo ArchInfo
}

func buildStockImage(image_name string) *StockImage {
	image := &StockImage{
		Name: image_name,
	}
	image.setInfo()
	image.setURL()
	return image
}

// ----------------------------------------------------------------------------
// Extract some info from the image filename
// - OS
// - Arch
// - Version
// ----------------------------------------------------------------------------

func (image *StockImage) setInfo() {
	info := ArchInfo{}
	os_match := regexp.MustCompile("alpine|debian")
	arch_match := regexp.MustCompile("arm64|aarch64")
	version_match := regexp.MustCompile(`\A[\d+\.]+`)
	for _, segment := range strings.Split(image.Name, "-") {
		switch {
		case os_match.MatchString(segment):
			info.OS = segment
		case arch_match.MatchString(segment):
			info.Arch = segment
		case version_match.MatchString(segment):
			info.Version = segment
		}
	}
	switch info.OS {
	case "alpine":
		info.Version = strings.Join(strings.Split(info.Version, ".")[:2], ".") // remove minor from version
	case "debian":
		if info.Version != "12" {
			panic(fmt.Sprintf("%s %s not supported", info.OS, info.Version))
		}
		info.Version = "bookworm"
	}
	image.ArchInfo = info
}

// ----------------------------------------------------------------------------
// URL where we'll download stock images, based on ArchInfo.OS
// ----------------------------------------------------------------------------

func (image *StockImage) setURL() {
	url := ""
	switch image.ArchInfo.OS {
	case "alpine":
		url = fmt.Sprintf("https://dl-cdn.alpinelinux.org/alpine/v%s/releases/cloud/%s.qcow2", image.ArchInfo.Version, image.Name)
	case "debian":
		url = fmt.Sprintf("https://cdimage.debian.org/cdimage/cloud/%s/latest/%s.qcow2", image.ArchInfo.Version, image.Name)
	default:
		panic(fmt.Sprintf("Unsupported OS %s", image.ArchInfo.OS))
	}
	image.URL = url
}

// ----------------------------------------------------------------------------
// Ask to download the stock image
// ----------------------------------------------------------------------------

func (image *StockImage) download() error {
	dst := path.Join("downloads", image.Name+".qcow2")
	if err := DownloadFile(dst, image.URL); err != nil {
		return fmt.Errorf("download failed : %w", err)
	}
	return nil
}
