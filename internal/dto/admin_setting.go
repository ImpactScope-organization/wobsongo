package dto

// UpdateAdminSettingRequest represents the request body for updating the admin
// settings of video discovery and creator monitoring. All fields are required
// because the dashboard submits the complete set of settings.
type UpdateAdminSettingRequest struct {
	// DiscoveryIntervalHours defines the interval between scheduled discovery runs in hours.
	DiscoveryIntervalHours int `json:"discovery_interval_hours" validate:"required,min=1,max=720"`

	// MaxResultsKeyword defines the maximum number of posts to retrieve per keyword.
	MaxResultsKeyword int `json:"max_results_keyword" validate:"required,min=1,max=100"`

	// MaxResultsInfluencer defines the maximum number of posts to retrieve per influencer.
	MaxResultsInfluencer int `json:"max_results_influencer" validate:"required,min=1,max=100"`

	// MaxCostPerRun defines the maximum allowed cost per run.
	MaxCostPerRun float64 `json:"max_cost_per_run" validate:"gt=0,lte=1000"`
}
