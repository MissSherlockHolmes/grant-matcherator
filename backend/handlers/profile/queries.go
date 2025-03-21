package profile

// [AI_QUERIES_START]
// QUERIES:
// {
//   "profile": {
//     "get_profile": "Retrieves user profile data",
//     "get_bio": "Retrieves user biographical data"
//   }
// }
// [AI_QUERIES_END]

const (
	// SelectProfileQuery retrieves a user's profile information
	SelectProfileQuery = `
		SELECT p.user_id, p.mission_statement, 
			array_to_json(p.sectors)::text, 
			array_to_json(p.target_groups)::text, 
			p.project_stage
		FROM matching_profiles p
		WHERE p.user_id = $1
	`

	// SelectBioQuery retrieves a user's biographical information
	SelectBioQuery = `
		SELECT p.user_id, p.location, p.website
		FROM profiles p
		WHERE p.user_id = $1
	`
)
