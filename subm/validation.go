package subm

type Validatable interface {
	IsValid() error
}

type SubmissionContent struct {
	Value string
}

func (subm *SubmissionContent) IsValid() error {
	const maxSubmissionLengthKilobytes = 64 // 64 KB
	if len(subm.Value) > maxSubmissionLengthKilobytes*1000 {
		return newErrSubmissionTooLong(maxSubmissionLengthKilobytes)
	}
	return nil
}

func (subm *SubmissionContent) String() string {
	return subm.Value
}
