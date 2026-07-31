package main

import (
	"strings"
	"testing"
)

func TestChatHTML_NotEmpty(t *testing.T) {
	if len(chatHTML) == 0 {
		t.Fatal("chatHTML should not be empty")
	}
	if !strings.HasPrefix(chatHTML, "<!DOCTYPE html>") {
		t.Error("chatHTML should start with <!DOCTYPE html>")
	}
}

func TestChatHTML_ContainsEssentialElements(t *testing.T) {
	essentialIDs := []string{
		`id="loginPage"`,
		`id="app"`,
		`id="sidebar"`,
		`id="messages"`,
		`id="adminPanel"`,
	}

	for _, id := range essentialIDs {
		if !strings.Contains(chatHTML, id) {
			t.Errorf("chatHTML missing required element ID: %s", id)
		}
	}
}

func TestChatHTML_EscFunctionHandlesQuotes(t *testing.T) {
	// Verify that esc() in JS handles quotes to prevent attribute breakout
	if strings.Contains(chatHTML, "function esc(s){const d=document.createElement('div');d.textContent=s;return d.innerHTML;}") {
		t.Error("esc() only uses textContent/innerHTML, which leaves double/single quotes unescaped in HTML attributes")
	}
}

func TestChatHTML_EditMsgUsesChildNodes(t *testing.T) {
	if strings.Contains(chatHTML, "bubble.textContent.replace('编辑删除','')") {
		t.Error("editMsg uses fragile textContent.replace('编辑删除','') which corrupts message content containing those words")
	}
}

