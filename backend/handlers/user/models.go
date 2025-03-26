package user

import "time"

// BasicUserResponse represents basic user information
type BasicUserResponse struct {
	ID                int     `json:"id"`
	OrganizationName  *string `json:"organization_name"`
	ProfilePictureURL *string `json:"profile_picture_url"`
}

// MatchingUser represents a user with matching-related information
type MatchingUser struct {
	ID                int      `json:"id"`
	Role              string   `json:"role"` // "provider" or "recipient"
	Email             string   `json:"email"`
	OrganizationName  *string  `json:"organization_name,omitempty"`
	ProfilePictureURL *string  `json:"profile_picture_url,omitempty"`
	MissionStatement  *string  `json:"mission_statement,omitempty"`
	State             *string  `json:"state,omitempty"`
	City              *string  `json:"city,omitempty"`
	ZIPCode           *string  `json:"zip_code,omitempty"`
	EIN               *string  `json:"ein,omitempty"`
	Language          *string  `json:"language,omitempty"`
	ApplicantType     *string  `json:"applicant_type,omitempty"`
	Sectors           []string `json:"sectors,omitempty"`
	TargetGroups      []string `json:"target_groups,omitempty"`
	ProjectStage      *string  `json:"project_stage,omitempty"`
	WebsiteURL        *string  `json:"website_url,omitempty"`
	ContactEmail      string   `json:"contact_email"`
	ChatOptIn         bool     `json:"chat_opt_in"`
	Location          *string  `json:"location,omitempty"`
	Description       *string  `json:"description,omitempty"`
}

// RecipientData represents recipient-specific information
type RecipientData struct {
	Needs           []string `json:"needs"`
	BudgetRequested float64  `json:"budget_requested"`
	TeamSize        int      `json:"team_size"`
	Timeline        string   `json:"timeline"`
	PriorFunding    bool     `json:"prior_funding"`
}

// ProviderData represents provider-specific information
type ProviderData struct {
	FundingType      string `json:"funding_type"`
	AmountOffered    string `json:"amount_offered"`
	RegionScope      string `json:"region_scope"`
	LocationNotes    string `json:"location_notes"`
	EligibilityNotes string `json:"eligibility_notes"`
	Deadline         string `json:"deadline"`
	ApplicationLink  string `json:"application_link"`
}

// User represents the core user entity
type User struct {
	ID        int       `json:"id"`
	Email     string    `json:"email"`
	Password  string    `json:"password,omitempty"`
	Role      string    `json:"role"` // "provider" or "recipient"
	CreatedAt time.Time `json:"created_at"`
}

// LoginResponse represents the response for login requests
type LoginResponse struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
	Token string `json:"token"`
}
