package networkmonitor

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/securestore"
)

// errRotationDryRun unwinds the dry-run transaction so its writes are
// discarded, which exercises the real update path without persisting it.
var errRotationDryRun = errors.New("secret rotation dry run")

// SecretRotationReport counts the credentials a rotation re-encrypted.
type SecretRotationReport struct {
	// Targets is the number of monitor targets with at least one credential.
	Targets int
	// Fields is the number of credential columns re-encrypted.
	Fields int
}

// RotateSNMPSecrets re-encrypts every stored SNMP credential from the old web
// secret to the new one across all tenants. The rows are rewritten in one
// transaction: a value that cannot be authenticated with the old secret aborts
// the rotation and leaves every ciphertext unchanged. When apply is false the
// updates are executed inside a transaction that is then rolled back, so a dry
// run proves the data can be re-encrypted without changing persisted state.
//
// Plaintext is only held for the duration of one Seal call and is cleared
// afterwards; errors name the target and column, never the credential.
func RotateSNMPSecrets(ctx context.Context, db *gorm.DB, oldSecret, newSecret string, apply bool) (SecretRotationReport, error) {
	var report SecretRotationReport
	if oldSecret == "" || newSecret == "" {
		return report, errors.New("both the old and the new web secret are required")
	}
	if oldSecret == newSecret {
		return report, errors.New("old and new web secret are identical")
	}
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var targets []domain.NetMonitorTarget
		if err := tx.Find(&targets).Error; err != nil {
			return fmt.Errorf("load monitor targets: %w", err)
		}
		for _, target := range targets {
			updates := make(map[string]any, 3)
			for _, field := range []struct {
				column     string
				ciphertext []byte
			}{
				{"snmp_community_encrypted", target.SNMPCommunityEncrypted},
				{"snmp_auth_encrypted", target.SNMPAuthEncrypted},
				{"snmp_privacy_encrypted", target.SNMPPrivacyEncrypted},
			} {
				if len(field.ciphertext) == 0 {
					continue
				}
				plaintext, err := securestore.Open(oldSecret, field.ciphertext)
				if err != nil {
					return fmt.Errorf("target %d: %s cannot be decrypted with the old secret: %w", target.ID, field.column, err)
				}
				sealed, sealErr := securestore.Seal(newSecret, plaintext)
				clear(plaintext)
				if sealErr != nil {
					return fmt.Errorf("target %d: re-encrypt %s: %w", target.ID, field.column, sealErr)
				}
				updates[field.column] = sealed
				report.Fields++
			}
			if len(updates) == 0 {
				continue
			}
			if err := tx.Model(&domain.NetMonitorTarget{}).Where("id = ?", target.ID).Updates(updates).Error; err != nil {
				return fmt.Errorf("target %d: store re-encrypted credentials: %w", target.ID, err)
			}
			report.Targets++
		}
		if !apply {
			return errRotationDryRun
		}
		return nil
	})
	if errors.Is(err, errRotationDryRun) {
		return report, nil
	}
	if err != nil {
		return SecretRotationReport{}, err
	}
	return report, nil
}
