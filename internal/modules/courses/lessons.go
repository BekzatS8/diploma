package courses

import (
	"sort"
	"strconv"
	"strings"
)

const defaultLessonKey = "default"

func BuildCourseLessons(materials []CourseMaterial) []CourseLesson {
	if len(materials) == 0 {
		return []CourseLesson{}
	}

	groups := make(map[string]*CourseLesson)
	order := make([]string, 0)
	for _, material := range materials {
		key := lessonKey(material.Metadata)
		title := lessonTitle(material.Metadata, key)
		position := lessonPosition(material.Metadata, material.SortOrder)

		group, ok := groups[key]
		if !ok {
			group = &CourseLesson{
				Key:      key,
				Title:    title,
				Position: position,
			}
			groups[key] = group
			order = append(order, key)
		}
		if position < group.Position {
			group.Position = position
		}
		group.Materials = append(group.Materials, material)
	}

	lessons := make([]CourseLesson, 0, len(order))
	for _, key := range order {
		lessons = append(lessons, *groups[key])
	}
	sort.SliceStable(lessons, func(i, j int) bool {
		if lessons[i].Position == lessons[j].Position {
			return lessons[i].Key < lessons[j].Key
		}
		return lessons[i].Position < lessons[j].Position
	})
	return lessons
}

func lessonKey(metadata map[string]any) string {
	key := metadataString(metadata, "lesson_key", "lesson_id", "section_key", "section_id")
	if key == "" {
		return defaultLessonKey
	}
	return key
}

func lessonTitle(metadata map[string]any, key string) string {
	title := metadataString(metadata, "lesson_title", "section_title")
	if title != "" {
		return title
	}
	if key == defaultLessonKey {
		return "Course materials"
	}
	return key
}

func lessonPosition(metadata map[string]any, fallback int) int {
	if position, ok := metadataInt(metadata, "lesson_position", "section_position"); ok {
		return position
	}
	return fallback
}

func metadataString(metadata map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := metadata[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case string:
			return strings.TrimSpace(typed)
		}
	}
	return ""
}

func metadataInt(metadata map[string]any, keys ...string) (int, bool) {
	for _, key := range keys {
		value, ok := metadata[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case int:
			return typed, true
		case int32:
			return int(typed), true
		case int64:
			return int(typed), true
		case float64:
			return int(typed), true
		case string:
			parsed, err := strconv.Atoi(strings.TrimSpace(typed))
			if err == nil {
				return parsed, true
			}
		}
	}
	return 0, false
}
