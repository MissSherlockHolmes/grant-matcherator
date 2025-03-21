package connection

// Connection queries
const (
	// GetConnectionsQuery retrieves all connections for a user
	GetConnectionsQuery = `
        SELECT 
            c.id,
            c.initiator_id,
            c.target_id,
            c.created_at,
            c.updated_at,
            CASE 
                WHEN c.initiator_id = $1 THEN p.name 
                ELSE p.name 
            END as other_user_name,
            CASE 
                WHEN c.initiator_id = $1 THEN p.profile_picture_url 
                ELSE p.profile_picture_url 
            END as other_user_picture,
            CASE 
                WHEN c.initiator_id = $1 THEN 'following' 
                ELSE 'follower' 
            END as connection_type
        FROM connections c
        JOIN profiles p ON 
            (c.initiator_id = $1 AND c.target_id = p.user_id) OR
            (c.target_id = $1 AND c.initiator_id = p.user_id)
        WHERE c.initiator_id = $1 OR c.target_id = $1
        ORDER BY c.created_at DESC
    `

	// GetPotentialMatchesQuery finds potential matches based on grant criteria
	GetPotentialMatchesQuery = `
        WITH provider_data AS (
            SELECT 
                u.id as provider_id,
                p.organization_name,
                p.sectors,
                p.target_groups,
                pd.funding_type,
                pd.amount_offered,
                pd.region_scope,
                pd.location_notes,
                pd.eligibility_notes,
                pd.deadline,
                pd.application_link
            FROM users u
            JOIN profiles p ON u.id = p.user_id
            JOIN provider_data pd ON u.id = pd.user_id
            WHERE u.role = 'provider'
        ),
        recipient_data AS (
            SELECT 
                u.id as recipient_id,
                p.organization_name,
                p.sectors,
                p.target_groups,
                p.project_stage,
                rd.needs,
                rd.budget_requested,
                rd.team_size,
                rd.timeline,
                rd.prior_funding
            FROM users u
            JOIN profiles p ON u.id = p.user_id
            JOIN recipient_data rd ON u.id = rd.user_id
            WHERE u.role = 'recipient'
        ),
        match_calculations AS (
            SELECT 
                p.provider_id,
                p.organization_name as provider_name,
                p.profile_picture_url as provider_picture,
                r.recipient_id,
                r.organization_name as recipient_name,
                r.profile_picture_url as recipient_picture,
                -- Sector alignment score (0-100)
                CASE 
                    WHEN p.sectors && r.sectors THEN (
                        CARDINALITY(ARRAY(
                            SELECT UNNEST(p.sectors) 
                            INTERSECT 
                            SELECT UNNEST(r.sectors)
                        ))::float / 
                        GREATEST(
                            CARDINALITY(p.sectors),
                            CARDINALITY(r.sectors)
                        )::float * 100
                    )
                    ELSE 0
                END as sector_score,
                -- Target group alignment score (0-100)
                CASE 
                    WHEN p.target_groups && r.target_groups THEN (
                        CARDINALITY(ARRAY(
                            SELECT UNNEST(p.target_groups) 
                            INTERSECT 
                            SELECT UNNEST(r.target_groups)
                        ))::float / 
                        GREATEST(
                            CARDINALITY(p.target_groups),
                            CARDINALITY(r.target_groups)
                        )::float * 100
                    )
                    ELSE 0
                END as target_group_score,
                -- Budget alignment score (0-100)
                CASE 
                    WHEN p.amount_offered >= r.budget_requested THEN 100
                    WHEN p.amount_offered >= r.budget_requested * 0.8 THEN 80
                    WHEN p.amount_offered >= r.budget_requested * 0.6 THEN 60
                    WHEN p.amount_offered >= r.budget_requested * 0.4 THEN 40
                    WHEN p.amount_offered >= r.budget_requested * 0.2 THEN 20
                    ELSE 0
                END as budget_score,
                -- Timeline alignment score (0-100)
                CASE 
                    WHEN p.deadline >= CURRENT_TIMESTAMP THEN
                        CASE 
                            WHEN r.timeline = 'immediate' AND p.deadline <= CURRENT_TIMESTAMP + INTERVAL '3 months' THEN 100
                            WHEN r.timeline = 'short_term' AND p.deadline <= CURRENT_TIMESTAMP + INTERVAL '6 months' THEN 100
                            WHEN r.timeline = 'medium_term' AND p.deadline <= CURRENT_TIMESTAMP + INTERVAL '12 months' THEN 100
                            WHEN r.timeline = 'long_term' AND p.deadline <= CURRENT_TIMESTAMP + INTERVAL '24 months' THEN 100
                            ELSE 50
                        END
                    ELSE 0
                END as timeline_score,
                -- Project stage alignment score (0-100)
                CASE 
                    WHEN r.project_stage = 'idea' AND p.funding_type = 'seed' THEN 100
                    WHEN r.project_stage = 'early_stage' AND p.funding_type = 'early' THEN 100
                    WHEN r.project_stage = 'growth' AND p.funding_type = 'growth' THEN 100
                    WHEN r.project_stage = 'scaling' AND p.funding_type = 'scale' THEN 100
                    ELSE 50
                END as stage_score
            FROM provider_data p
            CROSS JOIN recipient_data r
            WHERE NOT EXISTS (
                SELECT 1 FROM connections c
                WHERE (c.initiator_id = p.provider_id AND c.target_id = r.recipient_id)
                OR (c.initiator_id = r.recipient_id AND c.target_id = p.provider_id)
            )
        )
        SELECT 
            provider_id,
            provider_name,
            provider_picture,
            recipient_id,
            recipient_name,
            recipient_picture,
            sector_score,
            target_group_score,
            budget_score,
            timeline_score,
            stage_score,
            (
                sector_score * 0.30 +
                target_group_score * 0.30 +
                budget_score * 0.20 +
                timeline_score * 0.10 +
                stage_score * 0.10
            ) as match_score
        FROM match_calculations
        WHERE (
            sector_score * 0.30 +
            target_group_score * 0.30 +
            budget_score * 0.20 +
            timeline_score * 0.10 +
            stage_score * 0.10
        ) >= 60
        ORDER BY match_score DESC
        LIMIT 20
    `

	// CreateConnectionQuery creates a new connection
	CreateConnectionQuery = `
        INSERT INTO connections (initiator_id, target_id, created_at, updated_at)
        VALUES ($1, $2, NOW(), NOW())
        RETURNING id, created_at, updated_at
    `

	// DeleteConnectionQuery removes a connection
	DeleteConnectionQuery = `
        DELETE FROM connections 
        WHERE (initiator_id = $1 AND target_id = $2) OR
              (initiator_id = $2 AND target_id = $1)
    `

	// CheckConnectionExistsQuery checks if a connection already exists
	CheckConnectionExistsQuery = `
        SELECT EXISTS (
            SELECT 1 FROM connections 
            WHERE (initiator_id = $1 AND target_id = $2) OR
                  (initiator_id = $2 AND target_id = $1)
        )
    `
)
