package user

import "database/sql"

// IsUserAuthorized checks if a user can access another user's data
// Used by: GetUserHandler
// Dependencies: matches table, user_profiles
func IsUserAuthorized(db *sql.DB, requestingUserID int, targetUserID string) bool {
	var exists bool
	err := db.QueryRow(`
		WITH user_profiles AS (
			SELECT 
				u.role,
				p.sectors,
				p.target_groups,
				p.state,
				p.city,
				p.project_stage
			FROM users u
			LEFT JOIN matching_profiles p ON u.id = p.user_id
			WHERE u.id = $1 OR u.id = $2
		),
		requesting_user AS (
			SELECT * FROM user_profiles WHERE id = $1
		),
		target_user AS (
			SELECT * FROM user_profiles WHERE id = $2
		)
		SELECT EXISTS (
			SELECT 1 FROM matches 
			WHERE (user_id_1 = $1 AND user_id_2 = $2) OR (user_id_1 = $2 AND user_id_2 = $1)
			UNION
			SELECT 1 
			FROM requesting_user ru
			CROSS JOIN target_user tu
			LEFT JOIN matches m ON 
				(m.user_id_1 = $1 AND m.user_id_2 = $2) OR
				(m.user_id_2 = $1 AND m.user_id_1 = $2)
			WHERE ru.role != tu.role
			AND (
				-- Location match (if both have location data)
				(ru.state IS NOT NULL AND tu.state IS NOT NULL AND ru.state = tu.state AND ru.city = tu.city)
				OR
				-- Sector match (if both have sectors)
				(ru.sectors IS NOT NULL AND tu.sectors IS NOT NULL AND ru.sectors && tu.sectors)
				OR
				-- Target group match (if both have target groups)
				(ru.target_groups IS NOT NULL AND tu.target_groups IS NOT NULL AND ru.target_groups && tu.target_groups)
			)
			AND (
				m.id IS NULL 
				OR (m.status = 'pending' AND m.user_id_1 != $1)
			)
			AND NOT EXISTS (
				SELECT 1 FROM matches m2 
				WHERE m2.status = 'dismissed'
				AND (
					(m2.user_id_1 = $1 AND m2.user_id_2 = $2)
					OR 
					(m2.user_id_2 = $1 AND m2.user_id_1 = $2)
				)
			)
		)
	`, requestingUserID, targetUserID).Scan(&exists)

	if err != nil {
		return false // Treat DB errors as unauthorized
	}
	return exists
}
