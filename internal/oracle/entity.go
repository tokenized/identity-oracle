package oracle

import (
	"fmt"

	"github.com/tokenized/specification/dist/golang/actions"

	"github.com/pkg/errors"
)

// VerifyEntityIsSubset returns no error if all of the data in sub is specified in full.
// This is used to allow verification of some but not all of the known data about an identity.
func VerifyEntityIsSubset(sub, full *actions.EntityField) error {
	if len(sub.Name) != 0 && sub.Name != full.Name {
		return errors.New("Name doesn't match")
	}
	if len(sub.Type) != 0 && sub.Type != full.Type {
		return errors.New("Type doesn't match")
	}
	if len(sub.LEI) != 0 && sub.LEI != full.LEI {
		return errors.New("LEI doesn't match")
	}
	if len(sub.UnitNumber) != 0 && sub.UnitNumber != full.UnitNumber {
		return errors.New("UnitNumber doesn't match")
	}
	if len(sub.BuildingNumber) != 0 && sub.BuildingNumber != full.BuildingNumber {
		return errors.New("BuildingNumber doesn't match")
	}
	if len(sub.Street) != 0 && sub.Street != full.Street {
		return errors.New("Street doesn't match")
	}
	if len(sub.SuburbCity) != 0 && sub.SuburbCity != full.SuburbCity {
		return errors.New("SuburbCity doesn't match")
	}
	if len(sub.TerritoryStateProvinceCode) != 0 && sub.TerritoryStateProvinceCode != full.TerritoryStateProvinceCode {
		return errors.New("TerritoryStateProvinceCode doesn't match")
	}
	if len(sub.CountryCode) != 0 && sub.CountryCode != full.CountryCode {
		return errors.New("CountryCode doesn't match")
	}
	if len(sub.PostalZIPCode) != 0 && sub.PostalZIPCode != full.PostalZIPCode {
		return errors.New("PostalZIPCode doesn't match")
	}
	if len(sub.EmailAddress) != 0 && sub.EmailAddress != full.EmailAddress {
		return errors.New("EmailAddress doesn't match")
	}
	if len(sub.PhoneNumber) != 0 && sub.PhoneNumber != full.PhoneNumber {
		return errors.New("PhoneNumber doesn't match")
	}

	for i, admin := range sub.Administration {
		if !AdministratorIsInList(admin, full.Administration) {
			return fmt.Errorf("Administrator %d", i)
		}
	}

	for i, manager := range sub.Management {
		if !ManagerIsInList(manager, full.Management) {
			return fmt.Errorf("Manager %d", i)
		}
	}

	if len(sub.DomainName) != 0 && sub.DomainName != full.DomainName {
		return errors.New("DomainName doesn't match")
	}
	if len(sub.PaymailHandle) != 0 && sub.PaymailHandle != full.PaymailHandle {
		return errors.New("PaymailHandle doesn't match")
	}

	return nil
}

// AdministratorIsInList returns true if the administrator is in the list.
func AdministratorIsInList(admin *actions.AdministratorField, list []*actions.AdministratorField) bool {
	for _, item := range list {
		if admin.Type == item.Type && admin.Name == item.Name {
			return true // found match
		}
	}

	return false
}

// ManagerIsInList returns true if the manager is in the list.
func ManagerIsInList(manager *actions.ManagerField, list []*actions.ManagerField) bool {
	for _, item := range list {
		if manager.Type == item.Type && manager.Name == item.Name {
			return true // found match
		}
	}

	return false
}
