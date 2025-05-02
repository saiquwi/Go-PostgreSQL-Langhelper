package models

type UserLangs struct {
	ID     uint   `gorm:"primaryKey"`
	UserID uint   `gorm:"not null;index"`
	Lang1  string `gorm:"size:50"`
	Lang2  string `gorm:"size:50"`
	Lang3  string `gorm:"size:50"`
	Lang4  string `gorm:"size:50"`
	Lang5  string `gorm:"size:50"`

	User User `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}
