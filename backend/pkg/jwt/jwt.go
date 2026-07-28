package jwt

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/lexora/backend/internal/domain"
)

// token audiences
const (
	AudienceApp   = "app"
	AudienceAdmin = "admin"
)

type Signer struct {
	secret   []byte
	ttl      time.Duration
	audience string
}

func New(secret string, ttl time.Duration, audience string) *Signer {
	return &Signer{secret: []byte(secret), ttl: ttl, audience: audience}
}

type claims struct {
	OrgID      string `json:"org_id,omitempty"`
	SystemRole string `json:"sys_role"`
	OrgRole    string `json:"org_role,omitempty"`
	Name       string `json:"name,omitempty"`  // display only
	Email      string `json:"email,omitempty"` // display only
	jwt.RegisteredClaims
}

const issuer = "lexora"

func (s *Signer) Sign(id domain.Identity) (string, error) {
	now := time.Now()
	c := claims{
		SystemRole: id.SystemRole,
		OrgRole:    id.OrgRole,
		Name:       id.Name,
		Email:      id.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   id.UserID.String(),
			Issuer:    issuer,
			Audience:  jwt.ClaimStrings{s.audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
		},
	}
	if id.OrgID != uuid.Nil {
		c.OrgID = id.OrgID.String()
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(s.secret)
}

func (s *Signer) Verify(token string) (domain.Identity, error) {
	c := &claims{}
	_, err := jwt.ParseWithClaims(token, c, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, domain.ErrInvalidToken
		}
		return s.secret, nil
	}, jwt.WithIssuer(issuer), jwt.WithAudience(s.audience), jwt.WithExpirationRequired())
	if err != nil {
		return domain.Identity{}, domain.ErrInvalidToken
	}

	uid, err := uuid.Parse(c.Subject)
	if err != nil {
		return domain.Identity{}, domain.ErrInvalidToken
	}
	id := domain.Identity{UserID: uid, SystemRole: c.SystemRole, OrgRole: c.OrgRole}
	if c.OrgID != "" {
		oid, err := uuid.Parse(c.OrgID)
		if err != nil {
			return domain.Identity{}, domain.ErrInvalidToken
		}
		id.OrgID = oid
	}
	return id, nil
}
