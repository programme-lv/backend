package tasksrvc

import (
	"fmt"

	"github.com/programme-lv/backend/s3bucket"
)

type TaskService struct {
	s3PublicBucket   *s3bucket.S3Bucket
	s3TestfileBucket *s3bucket.S3Bucket
	s3TaskBucket     *s3bucket.S3Bucket
}

func NewTaskSrvc() (*TaskService, error) {
	publicBucket, err := s3bucket.NewS3Bucket("eu-central-1", "proglv-public")
	if err != nil {
		format := "failed to create S3 bucket: %w"
		return nil, fmt.Errorf(format, err)
	}
	testFileBucket, err := s3bucket.NewS3Bucket("eu-central-1", "proglv-tests")
	if err != nil {
		format := "failed to create S3 bucket: %w"
		return nil, fmt.Errorf(format, err)
	}
	taskBucket, err := s3bucket.NewS3Bucket("eu-central-1", "proglv-tasks")
	if err != nil {
		format := "failed to create S3 bucket: %w"
		return nil, fmt.Errorf(format, err)
	}

	return &TaskService{
		s3PublicBucket:   publicBucket,
		s3TestfileBucket: testFileBucket,
		s3TaskBucket:     taskBucket,
	}, nil
}
