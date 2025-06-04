package entity

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type (
	Element struct {
		gorm.Model
		AnswerID            uuid.UUID `gorm:"type:uuid" json:"answer_id"`
		QuestionOrderNumber uint      `gorm:"type:integer" json:"question_order_number"`
		Content             string    `gorm:"type:text" json:"content"`
		Answer              Answer    `gorm:"foreignKey:AnswerID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
	}

	Answer struct {
		ID        uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		FormID    uuid.UUID `gorm:"type:uuid" json:"form_id"`
		UserID    uuid.UUID `gorm:"type:uuid" json:"user_id"`
		IsComplete bool     `gorm:"default:false" json:"is_complete"`
		Elements  []Element `gorm:"foreignKey:AnswerID" json:"elements"`
	}
)

// BeforeCreate hook to generate UUID before creating Answer
func (a *Answer) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

// TableName returns the table name for Answer
func (Answer) TableName() string {
	return "answers"
}

// TableName returns the table name for Element
func (Element) TableName() string {
	return "answer_elements"
}

// GetElementByQuestionOrder returns an element by its question order number
func (a *Answer) GetElementByQuestionOrder(orderNumber uint) *Element {
	for i := range a.Elements {
		if a.Elements[i].QuestionOrderNumber == orderNumber {
			return &a.Elements[i]
		}
	}
	return nil
}

// AddElement adds a new element to the answer
func (a *Answer) AddElement(questionOrder uint, content string) {
	element := Element{
		AnswerID:            a.ID,
		QuestionOrderNumber: questionOrder,
		Content:             content,
	}
	a.Elements = append(a.Elements, element)
}

// MarkComplete marks the answer as complete
func (a *Answer) MarkComplete() {
	a.IsComplete = true
}

// IsAnswerComplete checks if the answer has all required elements
func (a *Answer) IsAnswerComplete() bool {
	return a.IsComplete && len(a.Elements) > 0
}

// GetElementsCount returns the number of elements in the answer
func (a *Answer) GetElementsCount() int {
	return len(a.Elements)
}

// Validate performs basic validation on the Answer
func (a *Answer) Validate() error {
	if a.FormID == uuid.Nil {
		return ErrInvalidFormID
	}
	if a.UserID == uuid.Nil {
		return ErrInvalidUserID
	}
	return nil
}

// Validate performs basic validation on the Element
func (e *Element) Validate() error {
	if e.AnswerID == uuid.Nil {
		return ErrInvalidAnswerID
	}
	if e.Content == "" {
		return ErrEmptyContent
	}
	return nil
}

// Custom errors
var (
	ErrInvalidFormID   = fmt.Errorf("invalid form ID")
	ErrInvalidUserID   = fmt.Errorf("invalid user ID")
	ErrInvalidAnswerID = fmt.Errorf("invalid answer ID")
	ErrEmptyContent    = fmt.Errorf("element content cannot be empty")
)
