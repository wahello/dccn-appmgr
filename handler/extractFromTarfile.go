package handler

import (
	"archive/tar"
	"bytes"
	"io"
)

func extractFromTarfile(tarf map[string]string, tarball *tar.Reader) error {
	count := len(tarf)
	for count > 0 {

		header, err := tarball.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if _, ok := tarf[header.Name]; ok {
			var b bytes.Buffer
			io.Copy(&b, tarball)
			tarf[header.Name] = string(b.Bytes())
			count--
		}
	}
	return nil
}
