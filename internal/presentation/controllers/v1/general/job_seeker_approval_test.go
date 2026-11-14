package general

import (
	"encoding/json"
	"testing"
	"time"
	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/enums"
	"zedlancer/backend/internal/domain/valueobjects"
)

func TestJobSeekerResponseApprovalFields(t *testing.T) {
	id := uint64(1)
	seeker := &entities.JobSeeker{BirthDate: valueobjects.JalaliDate{Year: time.Now().Year() - 621 - 16, Month: 1, Day: 1}, ParentID: &id, AdminReview: valueobjects.AdminReview{AdminApprovalStatus: enums.AdminReviewApproved}}
	for _, parentStatus := range []enums.AdminReviewStatus{enums.AdminReviewPending, enums.AdminReviewApproved, enums.AdminReviewRejected} {
		seeker.ParentAdminApprovalStatus = parentStatus
		data, err := json.Marshal(identityFromJobSeeker(seeker))
		if err != nil {
			t.Fatal(err)
		}
		var response map[string]any
		if err := json.Unmarshal(data, &response); err != nil {
			t.Fatal(err)
		}
		want := valueobjects.JobSeekerPending
		if parentStatus == enums.AdminReviewApproved {
			want = valueobjects.JobSeekerApproved
		}
		if response["signup_status"] != want || response["parent_connected"] != true || response["admin_approved"] != true || response["admin_approved_parent"] != (parentStatus == enums.AdminReviewApproved) {
			t.Fatalf("incorrect approval response: %s", data)
		}
		if _, exists := response["is_approved"]; exists {
			t.Fatal("obsolete is_approved field still returned")
		}
		if _, exists := response["signup_stage"]; exists {
			t.Fatal("internal signup stage still returned")
		}
	}
}
