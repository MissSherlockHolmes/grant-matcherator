package user

// User queries
const (
	// SelectUserQuery retrieves basic user information
	SelectUserQuery = `
		SELECT u.role, u.id, u.email, 
			p.organization_name, 
			p.website,
			p.contact_email,
			p.state,
			p.city,
			p.zip_code,
			p.ein,
			p.language,
			p.applicant_type,
			COALESCE(p.sectors, '{}'),
			COALESCE(p.target_groups, '{}'),
			p.project_stage,
			p.profile_picture_url
		FROM users u
		LEFT JOIN matching_profiles p ON u.id = p.user_id
		WHERE u.id = $1
	`

	// SelectRecipientQuery retrieves recipient-specific information
	SelectRecipientQuery = `
		SELECT needs, budget_requested,
			team_size, timeline, prior_funding
		FROM recipient_profiles
		WHERE user_id = $1
	`

	// SelectProviderQuery retrieves provider-specific information
	SelectProviderQuery = `
		SELECT funding_type, amount_offered, region_scope,
			location_notes, eligibility_notes, deadline,
			application_link
		FROM provider_profiles
		WHERE user_id = $1
	`
)
