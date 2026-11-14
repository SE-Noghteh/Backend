package enums

import "slices"

type LegalEntityType string

const (
	LegalLimitedLiability  LegalEntityType = "limited_liability"
	LegalPrivateJointStock LegalEntityType = "private_joint_stock"
	LegalPublicJointStock  LegalEntityType = "public_joint_stock"
	LegalCooperative       LegalEntityType = "cooperative"
	LegalInstitution       LegalEntityType = "institution"
	LegalOrganization      LegalEntityType = "organization"
)

func LegalEntityTypes() []LegalEntityType {
	return []LegalEntityType{LegalLimitedLiability, LegalPrivateJointStock, LegalPublicJointStock, LegalCooperative, LegalInstitution, LegalOrganization}
}

func (t LegalEntityType) Valid() bool { return slices.Contains(LegalEntityTypes(), t) }
