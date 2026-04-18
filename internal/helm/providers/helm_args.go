package providers

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
)

// RepoAddArgs returns the helm CLI arguments to add a standard HTTP repository.
// Returns an error for OCI providers (they need registry login, not repo add).
func RepoAddArgs(name, url, providerID string, creds map[string]string) ([]string, error) {
	args := []string{"repo", "add", name, url}
	switch providerID {
	case "harbor", "nexus":
		args = append(args, "--username", creds["username"], "--password", creds["password"])
	case "artifactory":
		args = append(args, "--username", creds["username"], "--password", creds["apiKey"])
	case "ecr":
		return nil, fmt.Errorf("ECR is an OCI provider — use RegistryLogin instead")
	default:
		return nil, fmt.Errorf("unknown provider %q", providerID)
	}
	return args, nil
}

// ECRLoginHost builds the ECR registry host from account ID and region.
func ECRLoginHost(creds map[string]string) string {
	return fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com", creds["accountId"], creds["region"])
}

// ECRToken exchanges AWS credentials for a short-lived Docker password
// using the AWS SDK for Go v2. No aws CLI required.
func ECRToken(ctx context.Context, creds map[string]string) (string, error) {
	cfg := aws.Config{
		Region: creds["region"],
		Credentials: credentials.NewStaticCredentialsProvider(
			creds["accessKeyId"],
			creds["secretAccessKey"],
			"",
		),
	}
	client := ecr.NewFromConfig(cfg)
	out, err := client.GetAuthorizationToken(ctx, &ecr.GetAuthorizationTokenInput{})
	if err != nil {
		return "", fmt.Errorf("ecr get-authorization-token: %w", err)
	}
	if len(out.AuthorizationData) == 0 {
		return "", fmt.Errorf("ecr returned no authorization data")
	}
	raw, err := base64.StdEncoding.DecodeString(aws.ToString(out.AuthorizationData[0].AuthorizationToken))
	if err != nil {
		return "", fmt.Errorf("decode ecr token: %w", err)
	}
	// token format is "AWS:<password>"
	parts := strings.SplitN(string(raw), ":", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("unexpected ecr token format")
	}
	return parts[1], nil
}
