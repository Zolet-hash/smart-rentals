package models

import (
	"time"
)

type Tenant struct {
	ID         uint   `json:"id"`
	TenantName string `json:"tenant_name"`
	PaymentNo  string `json:"payment_no"`
	UnitID     Unit   `json:"unit_id"` //FK -> units

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Property struct {
	ID           uint      `json:"id"`
	LandlordID   uint      `json:"landlord_id"` //FK userID users table
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	Location     string    `json:"location"`
	PropertyType string    `json:"property_type"`
	Vacancy      bool      `json:"vacancy"`
	TotalRent    int       `json:"total_rent"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Unit struct {
	ID         uint `json:"id"`
	PropertyID uint `json:"property_id"` //FK property_id -> properties

	UnitName  string    `json:"unit_name"`
	UnitType  string    `json:"unit_type"`
	UnitPrice int       `json:"unit_price"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Payment struct {
	ID            uint   `json:"id"`
	TenantID      uint   `json:"tenant_id"`
	UnitID        uint   `json:"unit_id"`
	Amount        int    `json:"amount"`
	PaymentMethod string `json:"payment_method"`

	PaymentDate time.Time `json:"payment_date"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Notification struct {
	ID       uint   `json:"id"`
	Message  string `json:"message"`
	IsRead   bool   `json:"is_read"`
	TenantID uint   `json:"tenant_id"`

	CreatedAt time.Time `json:"created_at"`
}

type Message struct {
	ID       uint   `json:"id"`
	Sender   string `json:"sender"`
	Receiver string `json:"receiver"`
	Content  string `json:"content"`

	CreatedAt time.Time `json:"created_at"`
}
