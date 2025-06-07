package mock

//go:generate go tool mockgen -package mock -destination mocks.go github.com/abicky/pd-shift/internal/pd Client
