package server

import (
	"encoding/hex"
	"github.com/getlantern/go-update"
)

func checksumForFile(file string) (checksumHex string, err error) {
	var checksum []byte
	if checksum, err = update.ChecksumForFile(file); err != nil {
		return "", err
	}
	checksumHex = hex.EncodeToString(checksum)
	return checksumHex, nil
}
