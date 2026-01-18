package app

import "testing"

func TestFormatLastMessage_InlineImageWithText(t *testing.T) {
	got := formatLastMessage(CachedLastMessage{
		FromUserID: "me",
		ToUserID:   "u2",
		Content:    "喜欢吗[20260104/image.jpg]",
		Type:       "text",
		Time:       "t",
	}, "me")
	if got != "我: 喜欢吗 [图片]" {
		t.Fatalf("got %q, want %q", got, "我: 喜欢吗 [图片]")
	}
}

func TestFormatLastMessage_EmojiTokenNotFile(t *testing.T) {
	got := formatLastMessage(CachedLastMessage{
		FromUserID: "me",
		ToUserID:   "u2",
		Content:    "[doge]",
		Type:       "text",
		Time:       "t",
	}, "me")
	if got != "我: [doge]" {
		t.Fatalf("got %q, want %q", got, "我: [doge]")
	}
}

func TestInferMessageType_InlineImage(t *testing.T) {
	if got := inferMessageType("喜欢吗[20260104/image.jpg]"); got != "image" {
		t.Fatalf("got %q, want %q", got, "image")
	}
}

func TestInferMessageType_EmojiIsText(t *testing.T) {
	if got := inferMessageType("[doge]"); got != "text" {
		t.Fatalf("got %q, want %q", got, "text")
	}
}
