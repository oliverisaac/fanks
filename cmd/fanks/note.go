package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/oliverisaac/fanks/types"
	"github.com/oliverisaac/fanks/views"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func newNoteForUser(prompt, content string, user types.User) types.Note {
	return types.Note{
		User:       user,
		Prompt:     prompt,
		IsUserNote: true,
		Content:    content,
		CreatedAt:  time.Now(),
	}
}

func createNote(db *gorm.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		user, ok := GetSessionUser(c)
		if !ok {
			return fmt.Errorf("You must be logged in to create a note")
		}

		content := c.FormValue("content")
		prompt := c.FormValue("prompt")
		note := newNoteForUser(prompt, content, user)

		if note.Content == "" {
			return render(c, 422, views.CreateNoteForm(note, prompt, fmt.Errorf("you cannot have an empty note")))
		}

		if err := db.Create(&note).Error; err != nil {
			err = errors.Wrap(err, "Saving note to db")
			logrus.Error(err)
			if prompt == "" {
				prompt = randomPrompt()
			}
			return render(c, 500, views.CreateNoteForm(note, prompt, err))
		}

		return render(c, 200, views.CreateNoteForm(note, randomPrompt(), nil))
	}
}

func randomPrompt() string {
	prompts := []string{
		"Today I am grateful for...",
		"What simple pleasure brought a smile to your face today?",
		"Name one person who made your day better. Why?",
		"What is a small detail you appreciate about your surroundings right now?",
		"What is something you learned today that you're grateful for?",
		"Think about a challenge you overcame. What are you grateful for in that experience?",
		"Today I am thankful for my ability to...",
		"What sound or sight are you grateful for today?",
		"What's one thing you have that you sometimes take for granted?",
		"Today's best moment was...",
		"Who is someone you are grateful to have in your life, and why?",
		"What about your own body or mind are you grateful for?",
		"What is a specific food or drink you enjoyed today?",
		"What about your home or living space are you grateful for?",
		"Think about a quiet moment you had. What did you appreciate about it?",
		"What skill or talent are you grateful to have?",
	}
	return RandomItem(prompts)
}

func RandomItem[T any](items []T) T {
	if len(items) == 0 {
		var zero T
		return zero
	}
	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)

	randomIndex := r.Intn(len(items))

	return items[randomIndex]
}

func deleteNote(db *gorm.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		user, ok := GetSessionUser(c)
		if !ok {
			return fmt.Errorf("You must be logged in to delete a note")
		}

		noteID := c.Param("id")

		var note types.Note
		if err := db.First(&note, noteID).Error; err != nil {
			return errors.Wrap(err, "getting note from db")
		}

		if note.UserID != user.ID {
			return fmt.Errorf("You are not authorized to delete this note")
		}

		if err := db.Delete(&note).Error; err != nil {
			return errors.Wrap(err, "deleting note from db")
		}

		return c.NoContent(200)
	}
}

func GetAllNotes(db *gorm.DB) ([]types.Note, error) {
	ret := []types.Note{}
	result := db.Preload("User").Order("created_at DESC").Limit(50).Find(&ret)
	if result.Error != nil {
		return nil, errors.Wrapf(result.Error, "Looking for all notes")
	}
	return ret, nil
}
