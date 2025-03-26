package user

// User queries
const (
	// SelectBasicUserQuery retrieves minimal user information
	SelectBasicUserQuery = `
		SELECT u.id, p.organization_name, p.profile_picture_url
		FROM users u
		LEFT JOIN profiles p ON u.id = p.user_id
		WHERE u.id = $1
	`

	// SelectUserQuery retrieves basic user information
	SelectUserQuery = `
		SELECT u.role, u.id, u.email, 
			p.organization_name, 
			p.profile_picture_url,
			p.mission_statement,
			p.state,
			p.city,
			p.zip_code,
			p.ein,
			p.language,
			p.applicant_type,
			COALESCE(p.sectors, '{}'),
			COALESCE(p.target_groups, '{}'),
			p.project_stage,
			p.website_url,
			p.contact_email,
			p.chat_opt_in,
			p.location,
		FROM users u
		LEFT JOIN profiles p ON u.id = p.user_id
		WHERE u.id = $1
	`

	// SelectRecipientQuery retrieves recipient-specific information
	SelectRecipientQuery = `
		SELECT needs, budget_requested,
			team_size, timeline, prior_funding
		FROM recipient_data
		WHERE user_id = $1
	`

	// SelectProviderQuery retrieves provider-specific information
	SelectProviderQuery = `
		SELECT funding_type, amount_offered, region_scope,
			location_notes, eligibility_notes, deadline,
			application_link
		FROM provider_data
		WHERE user_id = $1
	`
)
