/*
 * Copyright (C) 2019 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package clients

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"strings"
)

type HTTPClientErr struct {
	ErrMessage string
	RetCode    int
	RetMessage string
}

func (ucErr *HTTPClientErr) Error() string {
	return fmt.Sprintf("%s: %d: %s", ucErr.ErrMessage, ucErr.RetCode, ucErr.RetMessage)
}

func HTTPClient() *http.Client {
	return &http.Client{}
}

func HTTPClientTLSNoVerify() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
}

func HTTPClientWithCA(caCertificates []x509.Certificate) (*http.Client, error) {
	config := &tls.Config{
		InsecureSkipVerify: false,
		RootCAs:            GetCertPool(caCertificates),
	}
	tr := &http.Transport{TLSClientConfig: config}

	return &http.Client{Transport: tr}, nil
}

func ResolvePath(baseURL, path string) string {
	if baseURL == "" ||
		path == "" {
		return ""
	}
	if strings.HasSuffix(baseURL, "/") {
		baseURL = baseURL[:len(baseURL)-1]
	}
	if strings.HasPrefix(path, "/") {
		path = path[1:]
	}
	return baseURL + "/" + path
}

func GetCertPool(trustedCACerts []x509.Certificate) *x509.CertPool{
	rootCAs, _ := x509.SystemCertPool()
	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}
	for _, rootCA := range trustedCACerts {
		//This is needed since we provide reference to AddCert(). Otherwise it would append the
		//same address to the cert pool and the last cert appended will be the only one available
		tempStoreCA := rootCA
		rootCAs.AddCert(&tempStoreCA)
	}
	return rootCAs
}
