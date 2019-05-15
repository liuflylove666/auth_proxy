package types

import (
	log "github.com/Sirupsen/logrus"
	"github.com/contiv/auth_proxy/common/errors"
)

const (
	// TenantClaimKey is a prefix added to Claim keys in the
	// authorization or token object to represent tenants
	TenantClaimKey = "tenant:"

	// RoleClaimKey is a const string which represents highest
	// available role available to a principal in token object or
	// authorization db
	RoleClaimKey = "role"
)

// RoleType each role type is associated with a group and set of capabilities
type RoleType uint

// Set of pre-defined roles here
const (
	Admin   RoleType = iota // can perform any operation
	Ops                     // restricted to only assigned tenants
	Invalid                 // Invalid role, this needs to be the last role
)

// Tenant is a type to represent the name of the tenant
type Tenant string

// String returns the string representation of `RoleType`
func (role RoleType) String() string {
	switch role {
	case Ops:
		return "ops"
	case Admin:
		return "admin"
	default:
		log.Debug("Illegal role type")
		return ""
	}
}

// Role returns the `RoleType` of given string
func Role(roleStr string) (RoleType, error) {
	switch roleStr {
	case Admin.String():
		return Admin, nil
	case Ops.String():
		return Ops, nil
	default:
		log.Debugf("Unsupported role %q", roleStr)
		return Invalid, errors.ErrUnsupportedType
	}

}

// LocalUser information
//
// Fields:
//  UserName: of the user. Read only field. Must be unique.
//  FirstName: of the user
//  LastName: of the user
//  Password: of the user. Not stored anywhere. Used only for updates.
//  Disable: if authorizations for this local user is disabled.
//  PasswordHash: of the password string.
//
type LocalUser struct {
	Username     string `json:"username"`
	Password     string `json:"password,omitempty"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Disable      bool   `json:"disable"`
	PasswordHash []byte `json:"password_hash,omitempty"`
}

type LdapConfiguration struct {
	Server                 string `json:"server"`
	Port                   uint16 `json:"port"`
	BaseDN                 string `json:"base_dn"`
	ServiceAccountDN       string `json:"service_account_dn"`
	ServiceAccountPassword string `json:"service_account_password,omitempty"`
	StartTLS               bool   `json:"start_tls"`
	InsecureSkipVerify     bool   `json:"insecure_skip_verify"`
	TLSCertIssuedTo        string `json:"tls_cert_issued_to"`
}

//   StoreURL: URL of the key-value store
//
type KVStoreConfig struct {
	StoreURL    []string `json:"kvstore-url"`
	StoreDriver string `json:"kvstore-driver"`
	DbTLSCert    string      `json:"db-tls-cert"`
	DbTLSKey     string      `json:"db-tls-key"`
	DbTLSCa  	 string      `json:"db-tls-cacert"`

}

//
type WatchState struct {
	Curr State
	Prev State
}

//
type CommonState struct {
	StateDriver StateDriver `json:"-"`
	ID          string      `json:"id"`
}
