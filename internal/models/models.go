package models

import "time"

type Role string

const (
	RolePublic     Role = "public"
	RoleMaintainer Role = "maintainer"
	RoleAdmin      Role = "admin"
)

type User struct {
	ID        int64     `db:"id" json:"id"`
	Username  string    `db:"username" json:"username"`
	Email     string    `db:"email" json:"email"`
	Password  string    `db:"password_hash" json:"-"`
	Role      Role      `db:"role" json:"role"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type Package struct {
	ID            int64     `db:"id" json:"id"`
	Name          string    `db:"name" json:"name"`
	Description   string    `db:"description" json:"description"`
	CreatedBy     int64     `db:"created_by" json:"created_by"`
	TokenRequired bool      `db:"token_required" json:"token_required"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
}

type PackageVersion struct {
	ID           int64     `db:"id" json:"id"`
	PackageID    int64     `db:"package_id" json:"package_id"`
	Version      string    `db:"version" json:"version"`
	Metadata     string    `db:"metadata" json:"metadata"`
	ReleasedBy   int64     `db:"released_by" json:"released_by"`
	ReleasedAt   time.Time `db:"released_at" json:"released_at"`
	IsDeprecated bool      `db:"is_deprecated" json:"is_deprecated"`
}

type Artifact struct {
	ID               int64     `db:"id" json:"id"`
	PackageVersionID int64     `db:"package_version_id" json:"package_version_id"`
	BlobURL          string    `db:"blob_url" json:"blob_url"`
	Filename         string    `db:"filename" json:"filename"`
	SizeBytes        int64     `db:"size_bytes" json:"size_bytes"`
	CreatedAt        time.Time `db:"created_at" json:"created_at"`
}

type Vote struct {
	ID        int64     `db:"id" json:"id"`
	UserID    int64     `db:"user_id" json:"user_id"`
	PackageID int64     `db:"package_id" json:"package_id"`
	Value     int       `db:"value" json:"value"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type Comment struct {
	ID               int64     `db:"id" json:"id"`
	UserID           int64     `db:"user_id" json:"user_id"`
	PackageID        int64     `db:"package_id" json:"package_id"`
	PackageVersionID *int64    `db:"package_version_id" json:"package_version_id"`
	Body             string    `db:"body" json:"body"`
	CreatedAt        time.Time `db:"created_at" json:"created_at"`
}
