package observe

func defaultRegistry() map[string]Provider {
	return map[string]Provider{
		"git":    NewGitProvider(),
		"github": NewGitHubProvider(),
	}
}
