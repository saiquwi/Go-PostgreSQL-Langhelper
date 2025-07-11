package models

type DeckWord struct {
	ID     uint `gorm:"primaryKey"`
	DeckID uint `gorm:"not null;index"`
	WordID uint `gorm:"not null;index"`

	Deck Deck `gorm:"foreignKey:DeckID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Word Word `gorm:"foreignKey:WordID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}
