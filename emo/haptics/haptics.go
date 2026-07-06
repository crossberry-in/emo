// Package haptics implements haptic feedback for emo, inspired by expo-haptics.
//
// Provides three types of haptic feedback:
//
//   haptics.Impact(haptics.ImpactLight)     // light tap
//   haptics.Impact(haptics.ImpactMedium)    // medium tap
//   haptics.Impact(haptics.ImpactHeavy)     // heavy tap
//   haptics.Notification(haptics.Success)   // success pattern
//   haptics.Notification(haptics.Warning)   // warning pattern
//   haptics.Notification(haptics.Error)     // error pattern
//   haptics.Selection()                     // selection click
//
// In dev mode, haptic calls are forwarded to the device via the plugin
// transport. In production, the embedded runtime calls the Android Vibrator
// directly via JNI.
package haptics

import (
	"fmt"

	"github.com/emo-framework/emo/plugin"
)

// ImpactFeedbackStyle controls the intensity of an impact haptic.
type ImpactFeedbackStyle int

const (
	ImpactLight   ImpactFeedbackStyle = 1
	ImpactMedium  ImpactFeedbackStyle = 2
	ImpactHeavy   ImpactFeedbackStyle = 3
	ImpactRigid   ImpactFeedbackStyle = 4
	ImpactSoft    ImpactFeedbackStyle = 5
)

// NotificationFeedbackType controls the pattern of a notification haptic.
type NotificationFeedbackType int

const (
	Success NotificationFeedbackType = 1
	Warning NotificationFeedbackType = 2
	Error   NotificationFeedbackType = 3
)

// Impact triggers an impact haptic with the given style.
// In dev mode, this calls the device's vibrator via the plugin transport.
func Impact(style ImpactFeedbackStyle) {
	plugin.Invoke("haptics", "impact", map[string]any{
		"style": int(style),
	}, func(result any, err error) {
		if err != nil {
			fmt.Printf("[haptics] impact error: %v\n", err)
		}
	})
}

// Notification triggers a notification haptic with the given type.
// Each type (Success, Warning, Error) produces a distinct vibration pattern.
func Notification(typ NotificationFeedbackType) {
	plugin.Invoke("haptics", "notification", map[string]any{
		"type": int(typ),
	}, func(result any, err error) {
		if err != nil {
			fmt.Printf("[haptics] notification error: %v\n", err)
		}
	})
}

// Selection triggers a selection haptic (a light click).
// Use this when the user selects an item from a list or picker.
func Selection() {
	plugin.Invoke("haptics", "selection", nil, func(result any, err error) {
		if err != nil {
			fmt.Printf("[haptics] selection error: %v\n", err)
		}
	})
}

// Vibrate triggers a custom vibration pattern.
// duration is in milliseconds.
func Vibrate(duration int) {
	plugin.Invoke("haptics", "vibrate", map[string]any{
		"duration": duration,
	}, func(result any, err error) {
		if err != nil {
			fmt.Printf("[haptics] vibrate error: %v\n", err)
		}
	})
}
