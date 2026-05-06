package dirclient

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/agntcy/dir/client"
)

const deviceFlowTimeout = 5 * time.Minute

// EnsureOIDCToken checks the token cache for a valid token and returns it.
// If no valid token exists, it runs the OIDC device flow to obtain one,
// writing the device code URL and instructions to output.
func EnsureOIDCToken(ctx context.Context, issuer, clientID string, output io.Writer) (string, error) {
	cache := client.NewTokenCache()

	tok, err := cache.GetValidToken()
	if err != nil {
		return "", fmt.Errorf("reading token cache: %w", err)
	}
	if tok != nil {
		return tok.AccessToken, nil
	}

	result, err := client.OIDC.RunDeviceFlow(ctx, &client.DeviceFlowConfig{
		Issuer:   issuer,
		ClientID: clientID,
		Timeout:  deviceFlowTimeout,
		Output:   output,
	})
	if err != nil {
		return "", fmt.Errorf("device flow: %w", err)
	}

	cached := &client.CachedToken{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		TokenType:    result.TokenType,
		Provider:     "oidc",
		Issuer:       issuer,
		ExpiresAt:    result.ExpiresAt,
		User:         result.Name,
		UserID:       result.Subject,
		Email:        result.Email,
		CreatedAt:    time.Now().UTC(),
	}
	if err := cache.Save(cached); err != nil {
		return "", fmt.Errorf("saving token: %w", err)
	}

	return result.AccessToken, nil
}
