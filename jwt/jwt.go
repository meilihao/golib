// 备选: https://github.com/lestrrat-go/jwx
package jwt

import (
	"time"

	"github.com/go-jose/go-jose/v3"
	"github.com/go-jose/go-jose/v3/jwt"
)

var (
	jwtSecret []byte
	signer    jose.Signer
)

func Init(secret string) {
	jwtSecret = []byte(secret)
	signer, _ = jose.NewSigner(jose.SigningKey{Algorithm: jose.HS256, Key: jwtSecret}, &jose.SignerOptions{})
}

// GenerateToken generate tokens used for auth
func GenerateToken(token interface{}, claims *jwt.Claims) (string, error) {
	return jwt.Signed(signer).Claims(claims).Claims(token).CompactSerialize()
}

// ParseToken parsing token
func ParseToken(raw string, token interface{}) (*jwt.Claims, error) {
	tok, err := jwt.ParseSigned(raw)
	if err != nil {
		return nil, err
	}

	claims := &jwt.Claims{}
	if err = tok.Claims(jwtSecret, claims, token); err != nil {
		return nil, err
	}

	err = claims.Validate(jwt.Expected{
		Time: time.Now(),
	})
	if err != nil {
		return nil, err
	}

	return claims, nil
}
