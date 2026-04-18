package main

import (
	"fmt"
	"os"
	"strings"

	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	raw, _ := os.ReadFile(os.Args[1])
	fmt.Println("=== BEFORE ===")
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.Contains(line, "certificate") || strings.Contains(line, "client-key") {
			fmt.Println(strings.TrimRight(line, " "))
		}
	}

	cfg, err := clientcmd.Load(raw)
	if err != nil { fmt.Println("parse:", err); os.Exit(1) }

	for _, cluster := range cfg.Clusters {
		if cluster.CertificateAuthority != "" {
			data, err := os.ReadFile(cluster.CertificateAuthority)
			if err != nil { fmt.Println("read CA:", err); continue }
			cluster.CertificateAuthorityData = data
			cluster.CertificateAuthority = ""
		}
	}
	for _, u := range cfg.AuthInfos {
		if u.ClientCertificate != "" {
			data, _ := os.ReadFile(u.ClientCertificate)
			u.ClientCertificateData = data
			u.ClientCertificate = ""
		}
		if u.ClientKey != "" {
			data, _ := os.ReadFile(u.ClientKey)
			u.ClientKeyData = data
			u.ClientKey = ""
		}
	}
	out, err := clientcmd.Write(*cfg)
	if err != nil { fmt.Println("write:", err); os.Exit(1) }

	fmt.Println("\n=== AFTER ===")
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "certificate") || strings.Contains(line, "client-key") {
			l := strings.TrimRight(line, " ")
			if len(l) > 80 { l = l[:80] + "..." }
			fmt.Println(l)
		}
	}
	fmt.Println("\nTest helm with flattened kubeconfig:")
	os.WriteFile("/tmp/mk-flat-test.yaml", out, 0600)
	fmt.Println("Written to /tmp/mk-flat-test.yaml")
}
