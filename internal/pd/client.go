package pd

import (
	"context"

	"github.com/PagerDuty/go-pagerduty"
)

type Client interface {
	GetScheduleWithContext(ctx context.Context, id string, o pagerduty.GetScheduleOptions) (*pagerduty.Schedule, error)
}
