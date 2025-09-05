package wechatpay

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
)

func getTlsClient(MchPayCert, MchCertKey string) (*http.Client, error) {
	var tr *http.Transport
	// certs, err := tls.LoadX509KeyPair(MchPayCertFilePath, MchCertKeyFilePath)
	certs, err := tls.X509KeyPair(wrapCert(MchPayCert), wrapPrivateKey(MchCertKey))
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	// caData, err := ioutil.ReadFile(MchRootCaFilePath)
	// if err != nil {
	// 	return nil, err
	// }

	pool.AppendCertsFromPEM(wrapCert(Config.WechatPayRootCA))

	tr = &http.Transport{
		TLSClientConfig: &tls.Config{
			Certificates: []tls.Certificate{certs},
			RootCAs:      pool,
		},
	}

	return &http.Client{Transport: tr}, nil
}

func getTlsClientByCertData(MchPayCert, MchCertKey []byte) (*http.Client, error) {
	var tr *http.Transport
	// certs, err := tls.LoadX509KeyPair(MchPayCertFilePath, MchCertKeyFilePath)
	certs, err := tls.X509KeyPair(MchPayCert, MchCertKey)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	// caData, err := ioutil.ReadFile(MchRootCaFilePath)
	// if err != nil {
	// 	return nil, err
	// }

	pool.AppendCertsFromPEM(wrapCert(Config.WechatPayRootCA))

	tr = &http.Transport{
		TLSClientConfig: &tls.Config{
			Certificates: []tls.Certificate{certs},
			RootCAs:      pool,
		},
	}

	return &http.Client{Transport: tr}, nil
}

func wrapCert(cert string) []byte {
	return []byte(fmt.Sprintf(`
-----BEGIN CERTIFICATE-----
%s
-----END CERTIFICATE-----
	`, cert))
}

func wrapPrivateKey(key string) []byte {
	return []byte(fmt.Sprintf(`
-----BEGIN PRIVATE KEY-----
%s
-----END PRIVATE KEY-----
	`, key))
}
