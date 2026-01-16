package store

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"strings"

	"ebuild/internal/models"

	"github.com/jmoiron/sqlx"
)

type Store struct {
	DB *sqlx.DB
}

func New(db *sqlx.DB) *Store { return &Store{DB: db} }

func hashTokenRaw(t string) string {
	h := sha256.Sum256([]byte(t))
	return hex.EncodeToString(h[:])
}

func (s *Store) CreateUser(u *models.User, passwordHash string) (int64, error) {
	res, err := s.DB.Exec(`INSERT INTO users (username, email, password_hash, role) VALUES (?, ?, ?, ?)`, u.Username, u.Email, passwordHash, u.Role)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) GetUserByUsername(username string) (*models.User, error) {
	var u models.User
	if err := s.DB.Get(&u, `SELECT id, username, email, password_hash, role, created_at FROM users WHERE username = ?`, username); err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) IsTokenRevoked(rawToken string) (bool, error) {
	h := hashTokenRaw(rawToken)
	var revokedAt sql.NullTime
	if err := s.DB.Get(&revokedAt, `SELECT revoked_at FROM tokens WHERE token_hash = ?`, h); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return revokedAt.Valid, nil
}

func (s *Store) CreatePackage(p *models.Package) (int64, error) {
	res, err := s.DB.Exec(`INSERT INTO packages (name, description, created_by, token_required) VALUES (?, ?, ?, ?)`, p.Name, p.Description, p.CreatedBy, p.TokenRequired)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) CreateVersion(v *models.PackageVersion) (int64, error) {
	res, err := s.DB.Exec(`INSERT INTO package_versions (package_id, version, metadata, released_by, released_at, is_deprecated) VALUES (?, ?, ?, ?, ?, ?)`, v.PackageID, v.Version, v.Metadata, v.ReleasedBy, v.ReleasedAt, v.IsDeprecated)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) GetPackageByID(id int64) (*models.Package, error) {
	var p models.Package
	if err := s.DB.Get(&p, `SELECT id, name, description, created_by, token_required, created_at FROM packages WHERE id = ?`, id); err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Store) IsUserMaintainer(pkgID, userID int64) (bool, error) {
	// check package created_by
	var owner sql.NullInt64
	if err := s.DB.Get(&owner, `SELECT created_by FROM packages WHERE id = ?`, pkgID); err != nil {
		return false, err
	}
	if owner.Valid && owner.Int64 == userID {
		return true, nil
	}
	var exists int
	if err := s.DB.Get(&exists, `SELECT 1 FROM package_maintainers WHERE package_id = ? AND user_id = ? LIMIT 1`, pkgID, userID); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return exists == 1, nil
}

func (s *Store) CreateArtifact(packageVersionID int64, blobURL, filename string, sizeBytes int64) (int64, error) {
	res, err := s.DB.Exec(`INSERT INTO artifacts (package_version_id, blob_url, filename, size_bytes, created_at) VALUES (?, ?, ?, ?, datetime('now'))`, packageVersionID, blobURL, filename, sizeBytes)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) UpsertVote(userID, packageID int64, value int) error {
	_, err := s.DB.Exec(`INSERT INTO votes (user_id, package_id, value, created_at) VALUES (?, ?, ?, datetime('now')) ON CONFLICT(user_id, package_id) DO UPDATE SET value = excluded.value, created_at = datetime('now')`, userID, packageID, value)
	return err
}

func (s *Store) CreateComment(userID, packageID int64, packageVersionID *int64, body string) (int64, error) {
	res, err := s.DB.Exec(`INSERT INTO comments (user_id, package_id, package_version_id, body, created_at) VALUES (?, ?, ?, ?, datetime('now'))`, userID, packageID, packageVersionID, body)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) CreateToken(ownerUserID int64, tokenHash string, scopes []string, isGenerated bool) (int64, error) {
	res, err := s.DB.Exec(`INSERT INTO tokens (owner_user_id, token_hash, is_generated, scopes, created_at) VALUES (?, ?, ?, ?, datetime('now'))`, ownerUserID, tokenHash, boolToInt(isGenerated), strings.Join(scopes, ","))
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
