package encrypt

import "crypto/md5"

func Md5sum(input []byte) [16]byte {
	return md5.Sum(input)
}
