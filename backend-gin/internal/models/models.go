package models

import (
	"time"
)

type Users struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Email        string    `json:"email" gorm:"not null"`
	PasswordHash string    `json:"-"`
	Role         []Role    `json:"role"`
	PhoneNumber  string    `json:"phone_number" gorm:"not null"`
	CreatedAT    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Role struct {
	ID       uint   `json:"id" gorm:"primaryKey"`
	Admin    string `json:"admin"`
	Landlord string `json:"landlord"`
	Tenant   string `json:"tenant"`
}

type UserRole struct {
	UserRoleID uint `json:"user_role_id"`
	UserID     uint `json:"user_id"` //foreign key for Users
	RoleID     uint `json:"role_id"` //Foreign key for Roles
}

type Landlord struct {
	ID         int
	UserRoleID uint
	Tenants    []Tenant
	Properties []Property
}

type Tenant struct {
	ID int
	UserRoleID  int
}

type Property struct {
	ID           int          `json:"id" gorm:"primaryKey"`  //1
	Title        string       `json:"title" gorm:"not null"` //Qwetu residents
	Owner        string       `json:"owner"`                 //Emmaunel Sulwe
	Description  string       `json:"description"`           //modern student appartments
	Location     string       `json:"location"`              //westlands, nairobi
	PropertyType string       `json:"property_type"`         // appartments
	ProperyImage ProperyImage //A property can have several images
	Vacancy      bool         `json:"vacancy"`    //true/false
	Units        []Unit       `json:"units"`      // a property can have several units
	TotalRent    int          `json:"total_rent"` //200, 000
	CreatedAT    time.Time    `json:"created_at"` //25th june 2024
	UpdatedAt    time.Time    `json:"updated_at"` //2 jan 2026
}

type Unit struct {
	ID         int `json:"id" gorm:"primaryKey"`
	PropertyID int `json:"property_id" gorm:"foreignKey"`
	Property   Property
	TenantID   int
	Tenant     Tenant
	UnitName   string
	UnitType   string //Bedsitter,
	UnitPrice  int    //4500
	CreatedAT  time.Time
	UpdatedAt  time.Time
}

type ProperyImage struct {
	ID         int
	PropertyID int
	ImageURL   string
	CreatedAT  time.Time
	UpdatedAt  time.Time
}

type Lease struct {
	ID         uint
	UnitID     uint
	Unit       Unit
	TenantID   int
	Tenant     Tenant //relationship with Tenant
	LeaseStart time.Time
	Status     string
	LeaseEnd   time.Time
}

type Payment struct {
	ID            int
	UnitID        uint
	Unit          Unit
	Tenant        Tenant
	PaymentMethod string
}

type Notification struct {
	ID int
}

type Message struct {
	ID int
}
