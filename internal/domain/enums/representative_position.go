package enums

import "slices"

type RepresentativePosition string

const (
	RepresentativeCEO                 RepresentativePosition = "ceo"
	RepresentativeAuthorizedSignatory RepresentativePosition = "authorized_signatory"
	RepresentativeAttorney            RepresentativePosition = "attorney"
)

func RepresentativePositions() []RepresentativePosition {
	return []RepresentativePosition{RepresentativeCEO, RepresentativeAuthorizedSignatory, RepresentativeAttorney}
}

func (p RepresentativePosition) Valid() bool { return slices.Contains(RepresentativePositions(), p) }
