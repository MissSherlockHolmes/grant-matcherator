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
		SELECT 
			p.user_id,
			p.organization_name,
			p.profile_picture_url,
			p.mission_statement,
			p.state,
			p.city,
			p.zip_code,
			p.ein,
			p.language,
			p.applicant_type,
			array_to_json(COALESCE(p.sectors, '{}'))::text,
			array_to_json(COALESCE(p.target_groups, '{}'))::text,
			p.project_stage,
			p.website_url,
			p.contact_email,
			p.chat_opt_in,
			p.location,
			u.role,
			u.status
		FROM profiles p
		JOIN users u ON u.id = p.user_id
		WHERE p.user_id = $1
	`

	// SelectBioQuery retrieves a user's biographical information
	SelectBioQuery = `
		SELECT p.user_id, p.location, p.website_url
		FROM profiles p
		WHERE p.user_id = $1
	`
)
