package models

type DeckLang struct {
	ID     uint `gorm:"primaryKey"`
	DeckID uint `gorm:"not null;index"`
	LangID uint `gorm:"not null;index"`

	Deck     Deck     `gorm:"foreignKey:DeckID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	UserLang UserLang `gorm:"foreignKey:LangID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}
