package application

import (
	"time"

	"github.com/google/uuid"
)

type StageTransition struct {
	ID        uuid.UUID
	FromStage PipelineStage
	ToStage   PipelineStage
	ChangedBy uuid.UUID
	ChangedAt time.Time
	Reason    string
}

type RejectReason struct {
	Code    string
	Comment string
}

