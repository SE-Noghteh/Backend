package postgres

import (
	"testing"
	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/valueobjects"
)

func TestJobSeekerVideoMapping(t *testing.T) {
	for _, object := range []*valueobjects.StoredObject{nil, {Key: "job-seekers/random/video.mp4", ContentType: "video/mp4", Size: 123}} {
		seeker := &entities.JobSeeker{BirthDate: valueobjects.JalaliDate{Year: 1380, Month: 1, Day: 1}, VerificationVideo: object}
		model := jobSeekerToModel(seeker)
		result, err := jobSeekerFromModel(model)
		if err != nil {
			t.Fatal(err)
		}
		if object == nil {
			if result.VerificationVideo != nil {
				t.Fatal("legacy account gained a video reference")
			}
		} else if result.VerificationVideo == nil || *result.VerificationVideo != *object {
			t.Fatal("video metadata was lost in model mapping")
		}
	}
}
