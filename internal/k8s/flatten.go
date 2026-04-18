package k8s

import (
	"encoding/base64"
	"fmt"
	"os"

	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// FlattenKubeconfig parses a raw kubeconfig string, reads any certificate/key
// files referenced by path, embeds their contents as base64 data, and returns
// a clean YAML with no external file dependencies. This ensures helm and other
// CLI tools work correctly when the kubeconfig is written to a temp file.
func FlattenKubeconfig(raw string) (string, error) {
	cfg, err := clientcmd.Load([]byte(raw))
	if err != nil {
		return raw, fmt.Errorf("parse kubeconfig: %w", err)
	}

	for name, cluster := range cfg.Clusters {
		if err := inlineClusterCert(cluster); err != nil {
			return raw, fmt.Errorf("cluster %s: %w", name, err)
		}
	}
	for name, authInfo := range cfg.AuthInfos {
		if err := inlineUserCerts(authInfo); err != nil {
			return raw, fmt.Errorf("user %s: %w", name, err)
		}
	}

	out, err := clientcmd.Write(*cfg)
	if err != nil {
		return raw, fmt.Errorf("serialize kubeconfig: %w", err)
	}
	return string(out), nil
}

func readFile(path string) ([]byte, error) {
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return data, nil
}

func inlineClusterCert(c *clientcmdapi.Cluster) error {
	if c.CertificateAuthority == "" {
		return nil
	}
	data, err := readFile(c.CertificateAuthority)
	if err != nil {
		return err
	}
	c.CertificateAuthorityData = []byte(base64.StdEncoding.EncodeToString(data))
	// client-go stores raw PEM bytes in *Data fields, not base64 strings.
	// Use the raw PEM directly.
	c.CertificateAuthorityData = data
	c.CertificateAuthority = ""
	return nil
}

func inlineUserCerts(u *clientcmdapi.AuthInfo) error {
	if u.ClientCertificate != "" {
		data, err := readFile(u.ClientCertificate)
		if err != nil {
			return err
		}
		u.ClientCertificateData = data
		u.ClientCertificate = ""
	}
	if u.ClientKey != "" {
		data, err := readFile(u.ClientKey)
		if err != nil {
			return err
		}
		u.ClientKeyData = data
		u.ClientKey = ""
	}
	return nil
}
