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
                WHEN c.initiator_id = $1 THEN COALESCE(p.organization_name, '') 
                ELSE COALESCE(p.organization_name, '') 
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
        LEFT JOIN profiles p ON 
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
                p.profile_picture_url,
                p.state,
                p.city,
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
            WHERE u.role = 'provider' AND u.status = 'active'
        ),
        recipient_data AS (
            SELECT 
                u.id as recipient_id,
                p.organization_name,
                p.sectors,
                p.target_groups,
                p.project_stage,
                p.profile_picture_url,
                p.state,
                p.city,
                rd.needs,
                rd.budget_requested,
                rd.team_size,
                rd.timeline,
                rd.prior_funding
            FROM users u
            JOIN profiles p ON u.id = p.user_id
            JOIN recipient_data rd ON u.id = rd.user_id
            WHERE u.role = 'recipient' AND u.status = 'active'
        ),
        match_calculations AS (
            SELECT 
                p.provider_id,
                p.organization_name as provider_name,
                COALESCE(p.profile_picture_url, '') as provider_picture,
                r.recipient_id,
                r.organization_name as recipient_name,
                COALESCE(r.profile_picture_url, '') as recipient_picture,
                CASE 
                    WHEN p.sectors && r.sectors THEN 
                        (CARDINALITY(ARRAY(SELECT UNNEST(p.sectors) INTERSECT SELECT UNNEST(r.sectors)))::float / 
                        GREATEST(CARDINALITY(p.sectors), CARDINALITY(r.sectors))::float * 100)
                    ELSE 0 
                END as sector_score,
                CASE 
                    WHEN p.target_groups && r.target_groups THEN 
                        (CARDINALITY(ARRAY(SELECT UNNEST(p.target_groups) INTERSECT SELECT UNNEST(r.target_groups)))::float / 
                        GREATEST(CARDINALITY(p.target_groups), CARDINALITY(r.target_groups))::float * 100)
                    ELSE 0 
                END as target_group_score,
                CASE 
                    WHEN COALESCE(p.amount_offered, 0) >= COALESCE(r.budget_requested, 0) THEN 100
                    ELSE (COALESCE(p.amount_offered, 0)::float / NULLIF(COALESCE(r.budget_requested, 0), 0)::float * 100)
                END as budget_score,
                CASE 
                    WHEN COALESCE(p.deadline, CURRENT_TIMESTAMP + INTERVAL '1 year') >= 
                        CURRENT_TIMESTAMP + (
                            CASE 
                                WHEN r.timeline = '1-3 months' THEN INTERVAL '3 months'
                                WHEN r.timeline = '3-6 months' THEN INTERVAL '6 months'
                                WHEN r.timeline = '6-12 months' THEN INTERVAL '12 months'
                                ELSE INTERVAL '0 months'
                            END
                        ) THEN 100
                    ELSE 0 
                END as timeline_score,
                CASE 
                    WHEN r.project_stage = 'Early Stage' AND p.funding_type = 'seed' THEN 100
                    WHEN r.project_stage = 'Growth Stage' AND p.funding_type = 'series a' THEN 100
                    WHEN r.project_stage = 'Mature Stage' AND p.funding_type = 'series b' THEN 100
                    WHEN r.project_stage = 'Early Stage' AND p.funding_type = 'pitch comp' THEN 80
                    WHEN r.project_stage = 'Growth Stage' AND p.funding_type = 'pitch comp' THEN 60
                    WHEN r.project_stage = 'Mature Stage' AND p.funding_type = 'pitch comp' THEN 40
                    ELSE 20
                END as stage_score,
                CASE 
                    WHEN p.state = r.state AND p.city = r.city THEN 100
                    WHEN p.state = r.state THEN 50
                    ELSE 0
                END as location_score
            FROM provider_data p
            INNER JOIN recipient_data r ON true
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
            location_score,
            (sector_score * 0.3 + target_group_score * 0.3 + budget_score * 0.2 + timeline_score * 0.1 + stage_score * 0.1) as match_score
        FROM match_calculations
        WHERE 
            -- Require minimum match score
            (sector_score * 0.3 + target_group_score * 0.3 + budget_score * 0.2 + timeline_score * 0.1 + stage_score * 0.1) >= 30
            -- Require at least one sector or target group match
            AND (sector_score > 0 OR target_group_score > 0)
            -- Require location match if both have location data
            AND (
                (provider_id IN (SELECT provider_id FROM provider_data WHERE state IS NOT NULL AND city IS NOT NULL)
                AND recipient_id IN (SELECT recipient_id FROM recipient_data WHERE state IS NOT NULL AND city IS NOT NULL)
                AND location_score > 0)
                OR
                (provider_id NOT IN (SELECT provider_id FROM provider_data WHERE state IS NOT NULL AND city IS NOT NULL)
                OR recipient_id NOT IN (SELECT recipient_id FROM recipient_data WHERE state IS NOT NULL AND city IS NOT NULL))
            )
            -- Exclude users that are already connected
            AND NOT EXISTS (
                SELECT 1 FROM connections c
                WHERE (c.initiator_id = provider_id AND c.target_id = recipient_id)
                   OR (c.initiator_id = recipient_id AND c.target_id = provider_id)
            )
        ORDER BY match_score DESC
    `

	// CreateConnectionQuery creates a new connection
	CreateConnectionQuery = `
        INSERT INTO connections (initiator_id, target_id, connection_type, created_at, updated_at)
        VALUES ($1, $2, $3, NOW(), NOW())
        RETURNING id, created_at, updated_at
    `

	// DeleteConnectionQuery removes a connection
	DeleteConnectionQuery = `
        DELETE FROM connections 
        WHERE id = $1 AND (initiator_id = $2 OR target_id = $2)
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
