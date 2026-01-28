package models

import (
	"time"
)

type Tenant struct {
	ID         uint   `json:"id"`
	TenantName string `json:"tenant_name"`
	PaymentNo1 string `json:"payment_no1"` //1 -> 3
	PaymentNo2 string `json:"payment_no2"`
	Rent       string `json:"rent"`
	Balance    uint   `json:"balance"`
	UnitID     uint   `json:"unit_id"`     //FK -> units table
	LandlordID uint   `json:"landlord_id"` //FK -> users table

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Property struct {
	ID           uint      `json:"id"`
	LandlordID   uint      `json:"landlord_id"` //FK userID users table(landlord is a user)
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
	Vacancy   bool      `json:"vacancy"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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
