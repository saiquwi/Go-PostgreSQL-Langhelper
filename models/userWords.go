package models

type UserWords struct {
	ID     uint   `gorm:"primaryKey"`
	UserID uint   `gorm:"not null;index"`
	Tran1  string `gorm:"size:50"`
	Tran2  string `gorm:"size:50"`
	Tran3  string `gorm:"size:50"`
	Tran4  string `gorm:"size:50"`
	Tran5  string `gorm:"size:50"`

	User User `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}
