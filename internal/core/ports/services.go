package ports

type IPService interface {
	GetStats(domain string) (map[string]any, error)
}
