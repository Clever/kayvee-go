package router

// RuleMatchers describes which log lines a router rule applies to.
type RuleMatchers map[string][]string

// RuleOutput describes what to do if a log line matches a rule.
type RuleOutput map[string]interface{}

// Rule is a log routing rule
type Rule struct {
	Name     string
	Matchers RuleMatchers
	Output   RuleOutput
}

type metricsOutput struct {
	Series     string
	Dimensions []string
}

type alertOutput struct {
	Series     string
	Dimensions []string
	StatType   string `yaml:"stat_type"`
}

type analyticsOutput struct {
	Series string
}

type notificationOutput struct {
	Channel string
	Icon    string
	Message string
	User    string
}
