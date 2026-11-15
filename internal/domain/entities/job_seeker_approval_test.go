package entities

import (
	"testing"
	"time"
	"zedlancer/backend/internal/domain/enums"
	"zedlancer/backend/internal/domain/valueobjects"
)

func TestJobSeekerAcceptanceRequiresAgeAppropriateApprovals(t *testing.T) {
	birth := valueobjects.JalaliDate{Year: 1385, Month: 8, Day: 24}
	gregorian, err := birth.Gregorian()
	if err != nil {
		t.Fatal(err)
	}
	for _, age := range []int{15, 17, 18, 22} {
		for flags := 0; flags < 8; flags++ {
			seeker := &JobSeeker{BirthDate: birth}
			connected, admin, parentAdmin := flags&1 != 0, flags&2 != 0, flags&4 != 0
			if connected {
				id := uint64(1)
				seeker.ParentID = &id
			}
			if admin {
				seeker.AdminApprovalStatus = enums.AdminReviewApproved
			}
			if parentAdmin {
				seeker.ParentAdminApprovalStatus = enums.AdminReviewApproved
			}
			approval, err := seeker.ApprovalAt(gregorian.AddDate(age, 0, 0))
			if err != nil {
				t.Fatal(err)
			}
			accepted := admin && (age >= 18 || (connected && parentAdmin))
			if (approval.SignupStatus == valueobjects.JobSeekerApproved) != accepted || approval.ParentConnected != connected || approval.AdminApproved != admin || approval.AdminApprovedParent != (connected && parentAdmin) || approval.ParentApprovalRequired != (age < 18) {
				t.Fatalf("age %d, flags %d: %+v", age, flags, approval)
			}
		}
	}
}
func TestParentApprovalAndRejectionChangeMinorAcceptance(t *testing.T) {
	id := uint64(1)
	seeker := &JobSeeker{BirthDate: valueobjects.JalaliDate{Year: 1389, Month: 1, Day: 1}, ParentID: &id, AdminReview: valueobjects.AdminReview{AdminApprovalStatus: enums.AdminReviewApproved}}
	now := time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC)
	for _, status := range []enums.AdminReviewStatus{enums.AdminReviewPending, enums.AdminReviewApproved, enums.AdminReviewRejected} {
		seeker.ParentAdminApprovalStatus = status
		approval, err := seeker.ApprovalAt(now)
		if err != nil {
			t.Fatal(err)
		}
		if (approval.SignupStatus == valueobjects.JobSeekerApproved) != (status == enums.AdminReviewApproved) {
			t.Fatal("minor acceptance did not follow parent review", status, approval)
		}
	}
}
func TestAcceptanceAtEighteenthBirthdayForLeapDayBirth(t *testing.T) {
	birth, err := valueobjects.JalaliFromGregorian(time.Date(2008, 2, 29, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	seeker := &JobSeeker{BirthDate: birth, AdminReview: valueobjects.AdminReview{AdminApprovalStatus: enums.AdminReviewApproved}}
	before, _ := seeker.ApprovalAt(time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC))
	after, _ := seeker.ApprovalAt(time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC))
	if before.SignupStatus != valueobjects.JobSeekerPending || after.SignupStatus != valueobjects.JobSeekerApproved {
		t.Fatal("incorrect birthday boundary", before, after)
	}
}
