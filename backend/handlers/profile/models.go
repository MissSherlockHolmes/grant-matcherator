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
	ID               int      `json:"id"`
	MissionStatement string   `json:"mission_statement"`
	Sectors          []string `json:"sectors"`
	TargetGroups     []string `json:"target_groups"`
	ProjectStage     string   `json:"project_stage"`
}

// BioResponse represents the user's biographical data
type BioResponse struct {
	ID       int    `json:"id"`
	Location string `json:"location"`
	Website  string `json:"website"`
}
