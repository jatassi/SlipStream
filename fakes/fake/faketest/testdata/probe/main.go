package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
)

func main() {
	resp, err := http.Get(os.Args[1])
	if err != nil {
		if verr, ok := errors.AsType[*tls.CertificateVerificationError](err); ok {
			err = verr.Err
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("ok", string(body))
}
