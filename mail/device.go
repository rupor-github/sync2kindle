package mail

import (
	"fmt"
	"io"
	"mime"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gosimple/slug"
	"go.uber.org/zap"
	gomail "gopkg.in/gomail.v2"

	"sync2kindle/common"
	"sync2kindle/config"
	"sync2kindle/objects"
)

// should be usable in the zap log.Named()
const driverName = "e-mail"

type Device struct {
	log    *zap.Logger
	target string
	smtp   *config.SmtpConfig
	debug  bool
}

func Connect(target string, smtp *config.SmtpConfig, debug bool, log *zap.Logger) (*Device, error) {
	return &Device{target: target, smtp: smtp, debug: debug, log: log.Named(driverName)}, nil
}

// driver interface

func (d *Device) Disconnect() {
	// nothing to do at the moment
}

func (d *Device) Name() string {
	return driverName
}

// target path is destination e-mail address
// we will use from e-mail as unique id here instead of serial
func (d *Device) UniqueID() string {
	return d.smtp.From
}

func (d *Device) MkDir(obj *objects.ObjectInfo) error {
	d.log.Error("Action MkDir is not supported", zap.String("actor", d.Name()))
	return nil
}

func (d *Device) Remove(obj *objects.ObjectInfo) error {
	d.log.Error("Action Remove is not supported", zap.String("actor", d.Name()))
	return nil
}

const (
	safeTokenLength = 74
	rfc8187charset  = "UTF-8''"
)

func encodeParts(realname string) []string {
	part, parts := rfc8187charset, []string{}
	for _, sym := range realname {
		encoded := url.PathEscape(string(sym))
		if len(part)+len(encoded) > safeTokenLength {
			parts = append(parts, part)
			part = encoded
			continue
		}
		part += encoded
	}
	parts = append(parts, part)
	return parts
}

func encodeContentDispositionFilename(safename, realname string) string {
	res := `filename="` + safename + `"`
	for i, name := range encodeParts(realname) {
		res += fmt.Sprintf("; filename*%d*=%s", i, name)
	}
	return res
}

func (d *Device) Copy(obj *objects.ObjectInfo) (err error) {
	if obj == nil {
		panic("Copy is called with nil object")
	}

	defer func(start time.Time) {
		d.log.Debug("Executed action Copy", zap.String("actor", d.Name()), zap.Any("object", obj), zap.Duration("elapsed", time.Since(start)), zap.Error(err))
	}(time.Now())

	ext := filepath.Ext(obj.ObjectName)
	fullname := strings.TrimSuffix(filepath.Base(obj.ObjectName), ext)

	// For EPUB files, try to extract the title from metadata
	if strings.EqualFold(ext, ".epub") {
		if title, err := common.GetEPUBTitle(obj.ObjectName); err == nil && title != "" {
			fullname = title
			d.log.Debug("Using EPUB title metadata",
				zap.String("filename", filepath.Base(obj.ObjectName)),
				zap.String("title", title))
		} else {
			d.log.Debug("Failed to extract EPUB title, using filename",
				zap.String("filename", fullname),
				zap.Error(err))
		}
	}

	safename := slug.Make(fullname)

	m := gomail.NewMessage(gomail.SetCharset("UTF-8"), gomail.SetEncoding(gomail.Base64))
	m.SetAddressHeader("From", d.smtp.From, "sync2kindle")
	m.SetAddressHeader("To", d.target, "kindle device")
	m.SetHeader("Subject", "Sync to Kindle: "+fullname+ext)
	m.SetBody("text/plain", "This email has been sent by sync2kindle")
	m.Attach(obj.ObjectName,
		gomail.Rename(safename+ext),
		gomail.SetHeader(
			map[string][]string{
				"Content-Type":        {fmt.Sprintf(`%s; name="`, common.GetEMailContentType(ext)) + mime.BEncoding.Encode("UTF-8", fullname+ext) + `"`},
				"Content-Disposition": {`attachment; ` + encodeContentDispositionFilename(safename+ext, fullname+ext)},
			},
		),
	)

	if d.debug {
		var sf gomail.SendFunc = func(from string, to []string, m io.WriterTo) error {
			buf, err := os.Create(filepath.Join(d.smtp.Dir, safename+".mail"))
			if err != nil {
				return err
			}
			defer buf.Close()
			_, err = m.WriteTo(buf)
			if err != nil {
				return err
			}
			return nil
		}
		if err := gomail.Send(sf, m); err != nil {
			return err
		}
	}

	// real send
	if err := gomail.NewDialer(d.smtp.Server, d.smtp.Port, d.smtp.User, string(d.smtp.Password)).DialAndSend(m); err != nil {
		return fmt.Errorf("unable to send e-mail: %w", err)
	}
	return nil
}

func (d *Device) GetObjectInfos() (objects.ObjectInfoSet, error) {
	// always empty - it will be set outside, probably to history data, as we have no view into device state
	return objects.New(), nil
}
