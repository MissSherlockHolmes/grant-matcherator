package profile

// [AI_MODELS_START]
// MODELS:
// {
//   "ProfileResponse": {
//     "fields": ["ID", "MissionStatement", "Sectors", "TargetGroups", "ProjectStage"],
//     "json_tags": true,
//     "omitempty": false
//   },
//   "BioResponse": {
//     "fields": ["ID", "Location", "Website"],
//     "json_tags": true,
//     "omitempty": false
//   }
// }
// [AI_MODELS_END]

// ProfileResponse represents the user's "about me" information
type ProfileResponse struct {
	ID                int      `json:"id"`
	OrganizationName  string   `json:"organization_name"`
	ProfilePictureURL *string  `json:"profile_picture_url"`
	MissionStatement  string   `json:"mission_statement"`
	State             string   `json:"state"`
	City              string   `json:"city"`
	ZipCode           string   `json:"zip_code"`
	EIN               string   `json:"ein"`
	Language          string   `json:"language"`
	ApplicantType     string   `json:"applicant_type"`
	Sectors           []string `json:"sectors"`
	TargetGroups      []string `json:"target_groups"`
	ProjectStage      string   `json:"project_stage"`
	WebsiteURL        string   `json:"website_url"`
	ContactEmail      string   `json:"contact_email"`
	ChatOptIn         bool     `json:"chat_opt_in"`
	Location          string   `json:"location"`
	Role              string   `json:"role"`
	Status            string   `json:"status"`
}

// BioResponse represents the user's biographical data
type BioResponse struct {
	ID         int    `json:"id"`
	Location   string `json:"location"`
	WebsiteURL string `json:"website_url"`
}
