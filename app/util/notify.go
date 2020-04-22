package util

import (
	"fmt"
	"github.com/gen2brain/beeep"
)

func Notify(msg string) {
	err := beeep.Notify("", msg, "assets/icon.png")
	if err != nil {}
}

func Plural(n int, form1 string, form2 string, form5 string) string{
	// abs
	if n < 0 {
		n = -n
	}
	n1 := n % 10

	if n > 10 && n < 20 {
		return fmt.Sprintf("%d %s", n, form5)
	}
	if n1 > 1 && n1 < 5 {
		return fmt.Sprintf("%d %s", n, form2)
	}
	if n1 == 1 {
		return fmt.Sprintf("%d %s", n, form1)
	}
	return fmt.Sprintf("%d %s", n, form5)
}

func PluralMins(n int) string {
	return Plural(n, "минута", "минуты", "минут")
}

func PluralHours(n int) string {
	return Plural(n, "час", "часа", "часов")
}

func PluralTasks(n int) string {
	return Plural(n, "задача", "задачи", "задач")
}
