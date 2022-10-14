package internal

import (
	"encoding/base64"
	"os"
	"path"
)

func expandTLS(ts *tlsSettings) (err error) {
	if ts.Cert != "" {
		ts.cert, err = loadPem(ts.Cert)
		if err != nil {
			return err
		}
	}
	if ts.Key != "" {
		ts.key, err = loadPem(ts.Key)
		if err != nil {
			return err
		}
	}
	ts.customCAs = [][]byte{}
	for _, ca := range ts.CustomCA {
		cp, err := loadPem(ca)
		if err != nil {
			return err
		}
		ts.customCAs = append(ts.customCAs, cp)
	}
	return nil
}

func loadPem(value string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(value)
	if err == nil {
		return data, nil
	}
	return os.ReadFile(path.Clean(value))
}
