package models

type UserWord struct {
	ID          uint   `gorm:"primaryKey"`
	LangID      uint   `gorm:"not null;index"`
	WordID      uint   `gorm:"not null;index"`
	Translation string `gorm:"size:50"`

	UserLang UserLang `gorm:"foreignKey:LangID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Word     Word     `gorm:"foreignKey:WordID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}
