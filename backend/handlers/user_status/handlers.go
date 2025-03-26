package user_status

import (
	"database/sql"
	"strconv"
	"time"

	"github.com/lib/pq"
)

// UpdateUserStatus updates the status of a user based on their role and profile completion
func UpdateUserStatus(tx *sql.Tx, userID string) error {
	// Convert userID to int
	uid, err := strconv.Atoi(userID)
	if err != nil {
		return err
	}

	// Get user's role
	var role string
	err = tx.QueryRow("SELECT role FROM users WHERE id = $1", uid).Scan(&role)
	if err != nil {
		return err
	}

	var newStatus string
	if role == "provider" {
		// Check if provider's deadline has passed
		var deadline time.Time
		err = tx.QueryRow("SELECT deadline FROM provider_data WHERE user_id = $1", uid).Scan(&deadline)
		if err == sql.ErrNoRows {
			// No deadline = active
			newStatus = "active"
		} else if err != nil {
			return err
		} else if deadline.Before(time.Now()) {
			// Past deadline = inactive
			newStatus = "inactive"
		} else {
			// Future deadline = active
			newStatus = "active"
		}
	} else {
		// Check if recipient's profile is complete
		var profile struct {
			OrganizationName string
			Sectors          []string
			TargetGroups     []string
			State            string
			City             string
			ZipCode          string
		}
		err = tx.QueryRow(`
			SELECT 
				organization_name,
				sectors,
				target_groups,
				state,
				city,
				zip_code
			FROM profiles
			WHERE user_id = $1
		`, uid).Scan(
			&profile.OrganizationName,
			pq.Array(&profile.Sectors),
			pq.Array(&profile.TargetGroups),
			&profile.State,
			&profile.City,
			&profile.ZipCode,
		)
		if err == sql.ErrNoRows {
			newStatus = "inactive"
		} else if err != nil {
			return err
		} else if profile.OrganizationName != "" &&
			len(profile.Sectors) > 0 &&
			len(profile.TargetGroups) > 0 &&
			profile.State != "" &&
			profile.City != "" &&
			profile.ZipCode != "" {
			newStatus = "active"
		} else {
			newStatus = "inactive"
		}
	}

	// Update the user's status
	_, err = tx.Exec("UPDATE users SET status = $1 WHERE id = $2", newStatus, uid)
	return err
}
