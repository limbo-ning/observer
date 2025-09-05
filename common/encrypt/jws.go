package encrypt

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"log"
)

//https://github.com/golang-jwt/jwt/blob/v4.5.0/ecdsa.go

func getBitSize(signingMethod crypto.Hash) (int, error) {
	switch signingMethod {
	case crypto.SHA256:
		return 32, nil
	case crypto.SHA384:
		return 48, nil
	case crypto.SHA3_512:
		return 66, nil
	}

	return 0, errors.New("crypto not support")
}

func JWSECDSASign(signingMethod crypto.Hash, key []byte, data string) (string, error) {

	var block *pem.Block
	if block, _ = pem.Decode(key); block == nil {
		return "", errors.New("key not pem: " + string(key))
	}

	_, err := getBitSize(signingMethod)
	if err != nil {
		return "", err
	}

	hasher := signingMethod.New()
	hasher.Write([]byte(data))

	pkey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		p8key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return "", err
		}

		var ok bool
		if pkey, ok = p8key.(*ecdsa.PrivateKey); !ok {
			return "", errors.New("invalid private key")
		}
	}

	r, s, err := ecdsa.Sign(rand.Reader, pkey, hasher.Sum(nil))
	if err != nil {
		return "", err
	}

	curveBits := pkey.Curve.Params().BitSize
	log.Println("private key cur bit: ", curveBits)

	keyBytes := curveBits / 8
	if curveBits%8 > 0 {
		keyBytes += 1
	}

	// We serialize the outputs (r and s) into big-endian byte arrays
	// padded with zeros on the left to make sure the sizes work out.
	// Output must be 2*keyBytes long.
	out := make([]byte, 2*keyBytes)
	r.FillBytes(out[0:keyBytes]) // r is assigned to the first half of output.
	s.FillBytes(out[keyBytes:])  // s is assigned to the second half of output.

	return JWSEncode(out), nil
}

func JWSEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func JWSDecode(data string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(data)
}
