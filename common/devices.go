package common

import (
	"maps"
	"strings"
)

const (
	ThumbnailFolder = "system/thumbnails"
)

type SupportedProtocols int

const (
	ProtocolUSB SupportedProtocols = iota
	ProtocolMTP
	ProtocolMail
)

func (p SupportedProtocols) String() string {
	switch p {
	case ProtocolUSB:
		return "USB"
	case ProtocolMTP:
		return "MTP"
	case ProtocolMail:
		return "e-Mail"
	default:
		return "Unknown"
	}
}

var supportedFileFormatsForEMail = map[string]string{
	".DOC":  "application/msword",
	".DOCX": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".HTML": "text/html",
	".HTM":  "text/html",
	".RTF":  "application/rtf",
	".TXT":  "text/plain",
	".JPEG": "image/jpeg",
	".JPG":  "image/jpeg",
	".GIF":  "image/gif",
	".PNG":  "image/png",
	".BMP":  "image/bmp",
	".PDF":  "application/pdf",
	".EPUB": "application/epub+zip",
}

func IsSupportedEMailFormat(ext string) bool {
	for v := range maps.Keys(supportedFileFormatsForEMail) {
		if strings.EqualFold(v, ext) {
			return true
		}
	}
	return false
}

func GetEMailContentType(ext string) string {
	for k, v := range supportedFileFormatsForEMail {
		if strings.EqualFold(k, ext) {
			return v
		}
	}
	return "application/octet-stream"
}

var supportedDevices = []struct {
	vid, pid int
	protocol SupportedProtocols
}{
	{0x1949, 0x0002, ProtocolUSB}, // Kindle
	{0x1949, 0x0004, ProtocolUSB}, // Kindle 3/4/Paperwhite
	{0x1949, 0x9981, ProtocolMTP}, // So far this is true for Kindle Scribe and Kindle Paperwhite MTP devices
}

func IsKindleDevice(protocol SupportedProtocols, vid, pid int) bool {
	if vid == 0 && pid == 0 {
		return false
	}
	for _, d := range supportedDevices {
		if d.protocol == protocol && d.vid == vid && d.pid == pid {
			return true
		}
	}
	return false
}
